import { readFile, writeFile } from "node:fs/promises";

// Sourcey uses URL-safe slugs while MkDocs preserves index routes, case, and
// underscores. Keep the generated content intact and map those exceptional
// paths to the canonical URLs already served by the oastools documentation.
const canonicalPaths = new Map([
  ["/oastools/generator-beyond-boilerplate/", "/oastools/generator_beyond_boilerplate/"],
  ["/oastools/examples/index/", "/oastools/examples/"],
  ["/oastools/examples/workflows/index/", "/oastools/examples/workflows/"],
  ["/oastools/examples/programmatic-api/index/", "/oastools/examples/programmatic-api/"],
  ["/oastools/examples/walker/index/", "/oastools/examples/walker/"],
  ["/oastools/examples/petstore/index/", "/oastools/examples/petstore/"],
  ["/oastools/contributors/", "/oastools/CONTRIBUTORS/"],
  ["/oastools/license/", "/oastools/LICENSE/"],
]);

for (const name of ["llms.txt", "llms-full.txt"]) {
  let content = await readFile(new URL(`../.sourcey-dist/${name}`, import.meta.url), "utf8");

  for (const [generated, canonical] of canonicalPaths) {
    content = content.replaceAll(generated, canonical);
  }

  await writeFile(new URL(`../docs/${name}`, import.meta.url), content);
}
