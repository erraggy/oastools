#!/usr/bin/env bash
#
# conformance-vendor.sh materializes the vendored OpenAPI Specification
# conformance suite under testdata/conformance, at the commits pinned in
# testdata/conformance/sources.txt.
#
# With --update it first resolves each pinned ref to its current upstream head,
# vendors from there, and rewrites sources.txt with what it found. That is how a
# pin moves, and it is always a reviewed diff rather than a live fetch.
#
# Every failure is fatal. A partial vendor that looks like a successful run is
# the trap #391 and #401 were, so the fixture counts recorded in sources.txt are
# checked against what actually landed and a short tree exits non-zero.

set -euo pipefail

REPO_URL="https://github.com/OAI/OpenAPI-Specification.git"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$ROOT/testdata/conformance"
SOURCES="$DEST/sources.txt"

UPDATE=0
if [ "${1:-}" = "--update" ]; then
	UPDATE=1
elif [ $# -gt 0 ]; then
	echo "usage: $(basename "$0") [--update]" >&2
	exit 2
fi

if [ ! -f "$SOURCES" ]; then
	echo "error: $SOURCES not found" >&2
	exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# macOS ships bash 3.2, which has no associative arrays, so each phase re-reads
# the records from a file rather than holding them in a map.
records() {
	grep -vE '^[[:space:]]*(#|$)' "$1"
}

git init -q "$WORK/oai"
git -C "$WORK/oai" remote add origin "$REPO_URL"

# -----------------------------------------------------------------------------
# Phase 1: decide which commit each version is vendored from.
# -----------------------------------------------------------------------------
PLAN="$WORK/plan"
: >"$PLAN"

while read -r version ref commit subpath want_pass want_fail; do
	if [ "$UPDATE" -eq 1 ]; then
		head="$(git -C "$WORK/oai" ls-remote origin "refs/heads/$ref" | cut -f1)"
		if [ -z "$head" ]; then
			echo "error: $ref does not exist in $REPO_URL" >&2
			exit 1
		fi
		if [ "$head" != "$commit" ]; then
			echo "  $version: $ref moved ${commit:0:7} -> ${head:0:7}"
		fi
		commit="$head"
	fi
	printf '%s %s %s %s %s %s\n' "$version" "$ref" "$commit" "$subpath" "$want_pass" "$want_fail" >>"$PLAN"
done < <(records "$SOURCES")

# -----------------------------------------------------------------------------
# Phase 2: materialize each version and count what landed.
# -----------------------------------------------------------------------------
echo "Vendoring the OAI conformance suite into testdata/conformance..."
FOUND="$WORK/found"
: >"$FOUND"

while read -r version ref commit subpath want_pass want_fail; do
	echo "  ${version} (${ref} @ ${commit:0:7})"

	# Fetching the commit rather than the branch is what makes this a pinned
	# vendor: the result does not change when the branch moves.
	if ! git -C "$WORK/oai" fetch -q --depth 1 --filter=blob:none origin "$commit"; then
		echo "error: cannot fetch $commit from $REPO_URL" >&2
		exit 1
	fi

	rm -rf "${DEST:?}/$version"
	got_pass=0
	got_fail=0

	for kind in pass fail; do
		# A version that publishes no fixtures of this kind has no directory
		# upstream, so asking for one would fail the checkout.
		case "$kind" in
		pass) [ "$want_pass" -eq 0 ] && continue ;;
		fail) [ "$want_fail" -eq 0 ] && continue ;;
		esac

		rm -rf "${WORK:?}/oai/$subpath/$kind"
		if ! git -C "$WORK/oai" checkout -q FETCH_HEAD -- "$subpath/$kind"; then
			echo "error: $version: $subpath/$kind is absent at $commit" >&2
			exit 1
		fi

		mkdir -p "$DEST/$version/$kind"
		# Only the document fixtures are vendored. Upstream keeps its own
		# JavaScript harness beside them, and 3.3 keeps minimal-objects.yaml,
		# which is a table of object stubs rather than an OpenAPI document.
		find "$WORK/oai/$subpath/$kind" -maxdepth 1 -type f -name '*.yaml' \
			-exec cp {} "$DEST/$version/$kind/" \;

		n="$(find "$DEST/$version/$kind" -maxdepth 1 -type f -name '*.yaml' | wc -l | tr -d ' ')"
		echo "    $kind: $n"
		case "$kind" in
		pass) got_pass="$n" ;;
		fail) got_fail="$n" ;;
		esac
	done

	printf '%s %s %s %s %s %s\n' "$version" "$ref" "$commit" "$subpath" "$got_pass" "$got_fail" >>"$FOUND"
done < <(records "$PLAN")

# -----------------------------------------------------------------------------
# Phase 3: verify against the pins, or rewrite them.
# -----------------------------------------------------------------------------
if [ "$UPDATE" -eq 1 ]; then
	# The comment header is this file's documentation, so it is carried across
	# rather than regenerated.
	rewritten="$WORK/sources.txt"
	grep -E '^[[:space:]]*(#|$)' "$SOURCES" >"$rewritten"
	cat "$FOUND" >>"$rewritten"
	mv "$rewritten" "$SOURCES"
	echo "Updated sources.txt. Review 'git diff testdata/conformance' before committing."
	exit 0
fi

status=0
while read -r version ref commit subpath want_pass want_fail; do
	got="$(grep "^$version " "$FOUND")"
	got_pass="$(echo "$got" | cut -d' ' -f5)"
	got_fail="$(echo "$got" | cut -d' ' -f6)"
	if [ "$got_pass" -ne "$want_pass" ] || [ "$got_fail" -ne "$want_fail" ]; then
		echo "error: $version: sources.txt records ${want_pass}/${want_fail} pass/fail, upstream gave ${got_pass}/${got_fail}" >&2
		status=1
	fi
done < <(records "$SOURCES")

if [ "$status" -ne 0 ]; then
	echo "Vendoring failed: the fixture counts do not match sources.txt." >&2
	echo "Run 'make conformance-update' if upstream legitimately changed." >&2
	exit "$status"
fi

echo "Done. Pins unchanged, so 'git diff testdata/conformance' should be empty."
