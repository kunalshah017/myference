import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const publicDir = resolve(root, "web/public");
let svg = await readFile(resolve(publicDir, "og-image.svg"), "utf8");
let embedded = 0;

for (const logo of ["openai", "anthropic", "ollama", "kimi", "monad"]) {
  const png = await readFile(resolve(publicDir, `brand/providers/${logo}.png`));
  const reference = `href="brand/providers/${logo}.png"`;
  if (!svg.includes(reference)) throw new Error(`Missing ${logo} logo reference`);
  svg = svg.replace(
    reference,
    `href="data:image/png;base64,${png.toString("base64")}"`,
  );
  embedded += 1;
}

if (embedded !== 5 || (svg.match(/data:image\/png;base64,/g) ?? []).length !== 5) {
  throw new Error("Expected exactly five embedded social-card logos");
}

const output = resolve(process.argv[2] ?? "/tmp/myference-og-image-inline.svg");
await writeFile(output, svg);
console.log(`Inlined ${embedded} logos in ${output}`);
