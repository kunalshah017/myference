import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const publicDir = resolve(root, "web/public");

async function text(path) {
  return readFile(resolve(root, path), "utf8");
}

async function bytes(path) {
  return readFile(resolve(publicDir, path));
}

async function assertPng(path, width, height) {
  const file = await bytes(path);
  assert.equal(file.subarray(0, 8).toString("hex"), "89504e470d0a1a0a", `${path} is not PNG`);
  assert.equal(file.readUInt32BE(16), width, `${path} width`);
  assert.equal(file.readUInt32BE(20), height, `${path} height`);
}

const html = await text("web/index.html");
const requiredHtml = [
  'rel="icon" type="image/svg+xml" href="/favicon.svg"',
  'rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png"',
  'rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png"',
  'rel="shortcut icon" href="/favicon.ico"',
  'rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png"',
  'rel="mask-icon" href="/safari-pinned-tab.svg" color="#5140bd"',
  'rel="manifest" href="/site.webmanifest"',
  'rel="canonical" href="https://myference.xyz/"',
  'name="theme-color" content="#171522"',
  'name="msapplication-TileColor" content="#171522"',
  'property="og:type" content="website"',
  'property="og:site_name" content="Myference"',
  'property="og:url" content="https://myference.xyz/"',
  'property="og:title" content="Myference — distributed AI inference on Monad"',
  'property="og:description" content="Use hosted AI or turn unused compute into a paid inference provider, with native MON settlement."',
  'property="og:image" content="https://myference.xyz/og-image.png"',
  'property="og:image:width" content="1200"',
  'property="og:image:height" content="630"',
  'property="og:image:alt"',
  'name="twitter:card" content="summary_large_image"',
  'name="twitter:title" content="Myference — distributed AI inference on Monad"',
  'name="twitter:description" content="Use hosted AI or turn unused compute into a paid inference provider, with native MON settlement."',
  'name="twitter:image" content="https://myference.xyz/og-image.png"',
  'name="twitter:image:alt"',
];
for (const declaration of requiredHtml) {
  assert.ok(html.includes(declaration), `missing HTML metadata: ${declaration}`);
}
assert.ok(!html.includes("example.com"), "HTML contains a placeholder domain");
const browserConfig = await text("web/public/browserconfig.xml");
assert.ok(browserConfig.includes('/mstile-150x150.png'), "Windows tile is not configured");
assert.ok(browserConfig.includes("#171522"), "Windows tile color does not match the theme");

const manifest = JSON.parse(await text("web/public/site.webmanifest"));
assert.deepEqual(
  {
    name: manifest.name,
    short_name: manifest.short_name,
    start_url: manifest.start_url,
    display: manifest.display,
    background_color: manifest.background_color,
    theme_color: manifest.theme_color,
  },
  {
    name: "Myference",
    short_name: "Myference",
    start_url: "/",
    display: "standalone",
    background_color: "#fbfaff",
    theme_color: "#171522",
  },
);
assert.deepEqual(
  manifest.icons.map(({ src, sizes, type, purpose }) => ({ src, sizes, type, purpose })),
  [
    { src: "/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any maskable" },
    { src: "/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any maskable" },
  ],
);

await assertPng("favicon-16x16.png", 16, 16);
await assertPng("favicon-32x32.png", 32, 32);
await assertPng("apple-touch-icon.png", 180, 180);
await assertPng("mstile-150x150.png", 150, 150);
await assertPng("icon-192.png", 192, 192);
await assertPng("icon-512.png", 512, 512);
await assertPng("og-image.png", 1200, 630);

for (const svg of ["favicon.svg", "app-icon.svg", "safari-pinned-tab.svg", "og-image.svg"]) {
  const source = await readFile(resolve(publicDir, svg), "utf8");
  assert.match(source, /<svg\b/, `${svg} is not SVG`);
}
const appIconSource = await readFile(resolve(publicDir, "app-icon.svg"), "utf8");
assert.ok(appIconSource.includes('<rect width="64" height="64" fill="#171522"'), "installable app icon must have an opaque full-bleed background");
assert.ok(appIconSource.includes('id="maskable-safe-zone"'), "installable app mark must declare its maskable safe-zone group");
const socialSource = await readFile(resolve(publicDir, "og-image.svg"), "utf8");
assert.ok(socialSource.includes('id="safe-area" transform="translate(96 72)"'), "social card must keep a 96px horizontal safe area");
assert.ok(socialSource.includes('id="provider-icons"'), "social card must include the supported provider icons");
assert.ok(socialSource.includes('id="monad-settlement"'), "social card must include Monad settlement");
const socialSourceHash = createHash("sha256").update(socialSource);
for (const logo of ["openai", "anthropic", "ollama", "kimi", "monad"]) {
  assert.ok(socialSource.includes(`brand/providers/${logo}.png`), `social card is missing the ${logo} logo`);
  await assertPng(`brand/providers/${logo}.png`, 128, 128);
  socialSourceHash.update(await bytes(`brand/providers/${logo}.png`));
}
assert.ok(!socialSource.includes("LIVE ON MONAD"), "social card contains nonessential status copy");
assert.equal(socialSourceHash.digest("hex"), "e5ce8c529f64279af546a55c73345a573c90f92263300a849a52903a7c7bbc30", "social card sources changed; regenerate and visually review og-image.png");
assert.equal(createHash("sha256").update(await bytes("og-image.png")).digest("hex"), "0652af61c21bbf5386344934b1b6ccee1aca8351316606056267600aabf50156", "social card raster changed without updating its reviewed digest");

const ico = await bytes("favicon.ico");
assert.equal(ico.readUInt16LE(0), 0, "ICO reserved field");
assert.equal(ico.readUInt16LE(2), 1, "ICO image type");
assert.ok(ico.readUInt16LE(4) >= 1, "ICO image count");
assert.equal(ico.readUInt8(6), 32, "ICO first image width");
assert.equal(ico.readUInt8(7), 32, "ICO first image height");

console.log("Brand assets and metadata verified.");
