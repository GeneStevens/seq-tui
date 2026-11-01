// The browser build's smoke test: runs web/dist/seq-tui.wasm in Node with piped
// input and output (the page's globals faked), checks that the first screen is
// drawn at 80x24 from a recorded log, that a resize is taken, and that a key
// quits. Usage: node scripts/web-smoke.mjs [dist-dir] (make web-smoke).
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dist = path.resolve(process.argv[2] ?? path.join(root, "web", "dist"));
createRequire(import.meta.url)(path.join(dist, "wasm_exec.js")); // defines Go on globalThis

const out = [];
globalThis.seqTerm = { cols: 80, rows: 24, write: (u8) => out.push(Buffer.from(u8)) };
globalThis.seqReplay = readFileSync(path.join(root, "testdata", "logs", "play.jsonl"), "utf8");
let exited;
const done = new Promise((resolve) => { globalThis.seqExited = (msg) => { exited = msg; resolve(); }; });

const text = () => Buffer.concat(out).toString("utf8").replace(/\x1b\[[0-9;?<>=]*[ -\/]*[@-~]|\x1b[\]P][^\x07\x1b]*(\x07|\x1b\\)/g, "");
const until = async (what, ok, ms = 10000) => {
  const t0 = Date.now();
  while (!ok()) {
    if (Date.now() - t0 > ms) throw new Error(`timed out waiting for ${what}; output so far:\n${text().slice(-600)}`);
    await new Promise((r) => setTimeout(r, 20));
  }
};

const go = new Go();
const { instance } = await WebAssembly.instantiate(readFileSync(path.join(dist, "seq-tui.wasm")), go.importObject);
const running = go.run(instance);

await until("the page's input hook", () => typeof globalThis.seqInput === "function");
await until("the first screen", () => text().includes("HP"));
const first = text();
if (/too small/i.test(first)) throw new Error("drawn as too small at 80x24:\n" + first);
if (process.env.SMOKE_SHOW) console.log(first);
console.log(`first screen drawn: ${first.length} characters of output`);

globalThis.seqResize(100, 30);
const before = out.length;
await until("a redraw after the resize", () => out.length > before);
console.log("resize taken");

// PgUp scrolls the log panel back: the border says there is more below.
globalThis.seqInput("\x1b[5~");
await until("the log panel scrolled back", () => text().includes("· End"));
console.log("log panel scrolled back: ok");

globalThis.seqInput("q");
await Promise.race([done, until("the program to quit on q", () => false, 5000)]);
if (exited) throw new Error("ended with an error: " + exited);
await running;
console.log("quit on a key: ok");
process.exit(0); // wasm_exec throws when a Go timer fires after the exit
