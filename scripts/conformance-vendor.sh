#!/usr/bin/env bash
#
# conformance-vendor.sh materializes the vendored OpenAPI Specification
# conformance suite under testdata/conformance, at the commits pinned in
# testdata/conformance/sources.txt.
#
# With --update it resolves each pinned ref to its current upstream head,
# vendors from there, and rewrites sources.txt with what it found. That is how a
# pin moves, and it is always a reviewed diff rather than a live fetch.
#
# Every failure is fatal, and every check fails closed. A vendor that lands the
# wrong bytes must not exit zero: the counts and the per-version digest recorded
# in sources.txt are both compared against what is on disk, and anything the
# script cannot compare it refuses to guess at.

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
trap 'rm -rf "$WORK"' EXIT INT TERM

# sha256 of stdin, as bare hex. macOS ships shasum, most Linux images ship
# sha256sum, and CI may be either.
sha256() {
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 | cut -d' ' -f1
	else
		sha256sum | cut -d' ' -f1
	fi
}

# digest_version fingerprints one vendored version: every fixture's name and
# content, in a byte-stable order. Counts alone cannot see a fixture edited in
# place, nor one version's fixtures vendored into another's directory when both
# publish the same names, which 3.2 and 3.3 do for all 37 of their pass
# fixtures. internal/conformance recomputes this identically.
digest_version() {
	local vdir="$1" kind f
	{
		for kind in pass fail; do
			[ -d "$vdir/$kind" ] || continue
			for f in "$vdir/$kind"/*.yaml; do
				[ -e "$f" ] || continue
				printf '%s/%s %s\n' "$kind" "$(basename "$f")" "$(sha256 <"$f")"
			done
		done
	} | LC_ALL=C sort | sha256
}

# macOS ships bash 3.2, which has no associative arrays, so each phase re-reads
# the records from a file rather than holding them in a map.
records() {
	grep -vE '^[[:space:]]*(#|$)' "$1" || true
}

# A sources.txt that survives the existence check but carries no records would
# otherwise walk every loop zero times and report success, since the exit status
# of a process substitution is not observable to the shell.
record_count="$(records "$SOURCES" | wc -l | tr -d ' ')"
if [ "$record_count" -eq 0 ]; then
	echo "error: $SOURCES has no records" >&2
	exit 1
fi

git init -q "$WORK/oai"
git -C "$WORK/oai" remote add origin "$REPO_URL"

# -----------------------------------------------------------------------------
# Phase 1: decide which commit each version is vendored from.
# -----------------------------------------------------------------------------
PLAN="$WORK/plan"
: >"$PLAN"

while read -r version ref commit subpath want_pass want_fail want_digest; do
	# version names a directory that is about to be removed and rewritten, so it
	# is checked rather than trusted.
	case "$version" in
	*[!0-9.]* | *..* | "")
		echo "error: $version is not a usable version directory name" >&2
		exit 1
		;;
	esac

	if [ "$UPDATE" -eq 1 ]; then
		head="$(git -C "$WORK/oai" ls-remote origin "refs/heads/$ref" </dev/null | cut -f1)"
		if [ -z "$head" ]; then
			echo "error: $ref does not exist in $REPO_URL" >&2
			exit 1
		fi
		if [ "$head" != "$commit" ]; then
			echo "  $version: $ref moved ${commit:0:7} -> ${head:0:7}"
		fi
		commit="$head"
	fi
	printf '%s %s %s %s %s %s %s\n' \
		"$version" "$ref" "$commit" "$subpath" "$want_pass" "$want_fail" "$want_digest" >>"$PLAN"
done < <(records "$SOURCES")

if [ "$(wc -l <"$PLAN" | tr -d ' ')" -ne "$record_count" ]; then
	echo "error: planned $(wc -l <"$PLAN") versions from $record_count records" >&2
	exit 1
fi

# -----------------------------------------------------------------------------
# Phase 2: materialize each version and measure what landed.
# -----------------------------------------------------------------------------
echo "Vendoring the OAI conformance suite into testdata/conformance..."
FOUND="$WORK/found"
: >"$FOUND"

while read -r version ref commit subpath want_pass want_fail want_digest; do
	echo "  ${version} (${ref} @ ${commit:0:7})"

	# Fetching the commit rather than the branch is what makes this a pinned
	# vendor: the result does not change when the branch moves. Ordered before
	# the removal below so a network failure cannot destroy the existing tree.
	if ! git -C "$WORK/oai" fetch -q --depth 1 --filter=blob:none origin "$commit" </dev/null; then
		echo "error: cannot fetch $commit from $REPO_URL" >&2
		exit 1
	fi

	rm -rf "${DEST:?}/${version:?}"
	got_pass=0
	got_fail=0

	for kind in pass fail; do
		# The checkout is what establishes whether upstream publishes this kind,
		# rather than the recorded count, which would make a count of 0
		# self-perpetuating: a kind upstream later added would never be looked
		# for again.
		# Cleared first: every version branch checks out into the same working
		# path, so a stale tree would leave one version holding the union of
		# itself and the one vendored before it.
		rm -rf "${WORK:?}/oai/${subpath:?}/$kind"

		if ! git -C "$WORK/oai" checkout -q FETCH_HEAD -- "$subpath/$kind" </dev/null 2>/dev/null; then
			# Absent upstream. Legitimate when the pins record none of this kind,
			# and always legitimate under --update, which is re-deriving the
			# counts rather than trusting them. Compared as a string so a
			# malformed count cannot make the test error out and read as false.
			case "$kind" in
			pass) recorded="$want_pass" ;;
			fail) recorded="$want_fail" ;;
			esac
			if [ "$UPDATE" -eq 1 ] || [ "$recorded" = "0" ]; then
				continue
			fi
			echo "error: $version: cannot check out $subpath/$kind at $commit" >&2
			echo "  the path may not exist at that commit, or the blob fetch may have failed" >&2
			exit 1
		fi

		mkdir -p "$DEST/$version/$kind"
		# The copy is checked one file at a time: find does not report a failing
		# -exec in its own exit status, so an unwritable or full destination
		# would otherwise produce a short tree and a successful run.
		#
		# The .yaml filter narrows what a future upstream reshuffle could bring
		# in. It excludes nothing today: the checkout above materializes only
		# this directory, and upstream keeps its non-fixture files a level above.
		find "$WORK/oai/$subpath/$kind" -maxdepth 1 -type f -name '*.yaml' -print0 |
			while IFS= read -r -d '' f; do
				cp "$f" "$DEST/$version/$kind/" || {
					echo "error: $version/$kind: cannot copy $f" >&2
					exit 1
				}
			done || exit 1

		n="$(find "$DEST/$version/$kind" -maxdepth 1 -type f -name '*.yaml' -print0 | tr -dc '\0' | wc -c | tr -d ' ')"
		echo "    $kind: $n"
		case "$kind" in
		pass) got_pass="$n" ;;
		fail) got_fail="$n" ;;
		esac
	done

	# A version with no positive fixtures is a broken vendor rather than a
	# legitimate state: every version upstream publishes has some.
	if [ "$got_pass" -eq 0 ]; then
		echo "error: $version: no pass fixtures landed" >&2
		exit 1
	fi

	printf '%s %s %s %s %s %s %s\n' \
		"$version" "$ref" "$commit" "$subpath" "$got_pass" "$got_fail" "$(digest_version "$DEST/$version")" >>"$FOUND"
done < <(records "$PLAN")

if [ "$(wc -l <"$FOUND" | tr -d ' ')" -ne "$record_count" ]; then
	echo "error: vendored $(wc -l <"$FOUND") versions from $record_count records" >&2
	exit 1
fi

# -----------------------------------------------------------------------------
# Phase 3: verify against the pins, or rewrite them.
# -----------------------------------------------------------------------------
if [ "$UPDATE" -eq 1 ]; then
	# Staged inside the destination so the move is same-filesystem, and so an
	# interrupt cannot leave sources.txt truncated. A missing comment header is
	# not fatal: the records are what matter.
	rewritten="$(mktemp "$DEST/.sources.XXXXXX")"
	grep -E '^[[:space:]]*(#|$)' "$SOURCES" >"$rewritten" || true
	cat "$FOUND" >>"$rewritten"
	mv "$rewritten" "$SOURCES"
	echo "Updated sources.txt. Review 'git diff testdata/conformance' before committing."
	exit 0
fi

status=0
while read -r version ref commit subpath want_pass want_fail want_digest; do
	# Exact field match rather than a regex, and exactly one record, so a
	# duplicated or near-miss version cannot produce a multi-line value that
	# turns the comparisons below into a silent no-op.
	matches="$(awk -v v="$version" '$1 == v' "$FOUND" | wc -l | tr -d ' ')"
	if [ "$matches" -ne 1 ]; then
		echo "error: $version: $matches records in the vendored output, want exactly 1" >&2
		status=1
		continue
	fi
	got_pass="$(awk -v v="$version" '$1 == v {print $5}' "$FOUND")"
	got_fail="$(awk -v v="$version" '$1 == v {print $6}' "$FOUND")"
	got_digest="$(awk -v v="$version" '$1 == v {print $7}' "$FOUND")"

	# [ -ne ] returns 2 on a non-integer and, inside an if, that reads as "no
	# mismatch". Checked first so the comparison cannot fail open.
	case "$want_pass$want_fail$got_pass$got_fail" in
	*[!0-9]* | "")
		echo "error: $version: non-numeric fixture count" >&2
		status=1
		continue
		;;
	esac

	if [ "$got_pass" -ne "$want_pass" ] || [ "$got_fail" -ne "$want_fail" ]; then
		echo "error: $version: sources.txt records ${want_pass}/${want_fail} pass/fail, upstream gave ${got_pass}/${got_fail}" >&2
		status=1
	fi
	if [ -z "$want_digest" ] || [ "$got_digest" != "$want_digest" ]; then
		echo "error: $version: digest ${want_digest:-<missing>} recorded, ${got_digest} vendored" >&2
		echo "  the fixture names and counts may match while their contents do not" >&2
		status=1
	fi
done < <(records "$SOURCES")

if [ "$status" -ne 0 ]; then
	echo "Vendoring failed: the tree does not match sources.txt." >&2
	echo "Run 'make conformance-update' if upstream legitimately changed." >&2
	exit "$status"
fi

echo "Done. Pins unchanged, so the tree matches what sources.txt records."
