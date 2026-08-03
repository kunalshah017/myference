import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const publicDir = resolve(import.meta.dirname, "../web/public");
const png = await readFile(resolve(publicDir, "favicon-32x32.png"));
const header = Buffer.alloc(22);

header.writeUInt16LE(1, 2);
header.writeUInt16LE(1, 4);
header.writeUInt8(32, 6);
header.writeUInt8(32, 7);
header.writeUInt16LE(1, 10);
header.writeUInt16LE(32, 12);
header.writeUInt32LE(png.length, 14);
header.writeUInt32LE(header.length, 18);

await writeFile(resolve(publicDir, "favicon.ico"), Buffer.concat([header, png]));
console.log("Created web/public/favicon.ico");
