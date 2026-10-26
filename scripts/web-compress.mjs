// Writes a .gz and a .br beside each file in web/dist that a server sends
// compressed (the wasm, scripts, styles, fonts), for a server that serves
// precompressed files (Caddy's `file_server { precompressed br gzip }`), and
// prints the sizes. Usage: node scripts/web-compress.mjs <dist-dir>
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import path from "node:path";
import zlib from "node:zlib";

const dist = path.resolve(process.argv[2] ?? "web/dist");
const wanted = /\.(wasm|js|css|html|json|ttf)$/;
const walk = (d) => readdirSync(d).flatMap((n) => {
  const p = path.join(d, n);
  return statSync(p).isDirectory() ? walk(p) : [p];
});

let rows = [];
for (const file of walk(dist).filter((f) => wanted.test(f))) {
  const raw = readFileSync(file);
  const gz = zlib.gzipSync(raw, { level: 9 });
  const br = zlib.brotliCompressSync(raw, { params: {
    [zlib.constants.BROTLI_PARAM_QUALITY]: 11,
    [zlib.constants.BROTLI_PARAM_SIZE_HINT]: raw.length,
  } });
  writeFileSync(file + ".gz", gz);
  writeFileSync(file + ".br", br);
  rows.push([path.relative(dist, file), raw.length, gz.length, br.length]);
}
const kb = (n) => (n / 1024).toFixed(0).padStart(7) + " KB";
console.log("file".padEnd(34) + "    raw".padStart(10) + "     gzip".padStart(10) + "   brotli".padStart(10));
for (const [f, a, b, c] of rows.sort((x, y) => y[1] - x[1])) console.log(f.padEnd(34) + kb(a) + kb(b) + kb(c));
