// The browser display check: runs the webtest wasm in Node, feeds what it writes
// into a headless xterm.js (scripts/vendor/xterm-headless.js, 6.0.0) at 120x40
// the way the page does, and compares the terminal's rows with the frame the
// program meant to draw (seqLastView, set by the harness in harness_js.go).
// Cases: the picker, after Down twice, a screen change (Ctrl+L), the replayed
// game screen, the game after a few keys, and a resize to 100x30.
// Usage: node scripts/web-check.mjs <dist-dir> [--convert-eol=on|off] [--expect-fail]
// The page sets convertEol (web/app.js); --convert-eol=off shows the drift it
// prevents, and --expect-fail makes that the passing result.
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);
const dist = path.resolve(args.find((a) => !a.startsWith("--")) ?? path.join(root, "tmp", "websmoke"));
const convertEol = !args.includes("--convert-eol=off");
const expectFail = args.includes("--expect-fail");
const mode = args.find((a) => a.startsWith("--mode="))?.slice(7);

if (!mode) { // one wasm run per mode, each in its own process
  const { spawnSync } = await import("node:child_process");
  let bad = 0;
  // The page's xterm.js options for Option-click (D109): the headless terminal has
  // no mouse of its own to click, so the settings that let Option-click through
  // (and keep Shift-click selecting) are checked in the page's source.
  const page = readFileSync(path.join(root, "web", "app.js"), "utf8");
  for (const [what, re] of [
    ["macOptionClickForcesSelection: false (Option-click reaches the game)", /macOptionClickForcesSelection:\s*false/],
    ["altClickMovesCursor: false", /altClickMovesCursor:\s*false/],
    ["Shift forces a selection on a Mac too", /shouldForceSelection\s*=\s*\(e\)\s*=>\s*e\.shiftKey/],
  ]) {
    const ok = re.test(page);
    console.log(`  ${ok ? "ok  " : "FAIL"} page options: ${what}`);
    bad += ok ? 0 : 1;
  }
  for (const m of ["picker", "game"]) {
    const r = spawnSync(process.execPath, [fileURLToPath(import.meta.url), ...args, "--mode=" + m], { stdio: "inherit" });
    bad += r.status === 0 ? 0 : 1;
  }
  process.exit(bad ? 1 : 0);
}

const require = createRequire(import.meta.url);
const { Terminal } = require("./vendor/xterm-headless.js");
require(path.join(dist, "wasm_exec.js"));

const term = new Terminal({ cols: 120, rows: 40, scrollback: 0, convertEol, allowProposedApi: true });
let pending = 0, last = Date.now();
globalThis.seqTerm = { cols: 120, rows: 40, write: (u8) => { pending++; last = Date.now(); term.write(Buffer.from(u8), () => pending--); } };
globalThis.seqReplay = readFileSync(path.join(root, "testdata", "logs", "play.jsonl"), "utf8");
globalThis.seqHarness = mode;
globalThis.seqExited = (m) => { if (m) console.log("exited:", m); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const settle = async () => {
  const t0 = Date.now();
  await sleep(50);
  while ((pending || Date.now() - last < 300) && Date.now() - t0 < 3000) await sleep(25);
  while (pending) await sleep(10);
};
// The status line's "heard 17.8s ago" counts up between the frame and the read.
const clock = (s) => s.replace(/heard [0-9.]+m?s ago/g, "heard ~ ago");
const strip = (s) => s.replace(/\x1b\[[0-9;:?<>=]*[ -\/]*[@-~]|\x1b[\]P_][^\x07\x1b]*(\x07|\x1b\\)|\x1b[()][0-9A-Z]/g, "");

let failed = 0;
function compare(label) {
  const want = strip(globalThis.seqLastView || "").split("\n").map((l) => clock(l.trimEnd()));
  const buf = term.buffer.active;
  const bad = [];
  for (let y = 0; y < term.rows; y++) {
    const got = clock(buf.getLine(y).translateToString(true).trimEnd());
    if ((want[y] ?? "") !== got) bad.push({ y, want: want[y] ?? "", got });
  }
  if (bad.length) failed++;
  console.log(`  ${bad.length ? "DIFF" : "ok  "} ${mode}: ${label}: ${bad.length} of ${term.rows} rows differ`);
  for (const b of bad.slice(0, 2)) console.log(`    row ${b.y}\n      want: ${JSON.stringify(b.want.slice(0, 100))}\n      got:  ${JSON.stringify(b.got.slice(0, 100))}`);
}

const go = new Go();
const { instance } = await WebAssembly.instantiate(readFileSync(path.join(dist, "seq-tui.wasm")), go.importObject);
go.run(instance);
const t0 = Date.now();
while (typeof globalThis.seqInput !== "function") {
  if (Date.now() - t0 > 10000) { console.log("timed out waiting for the program"); process.exit(1); }
  await sleep(20);
}
await settle();
compare(mode === "picker" ? "the picker" : "the replayed game screen");
if (mode === "picker") {
  for (const n of [1, 2]) { globalThis.seqInput("\x1b[B"); await settle(); compare(`after Down x${n}`); }
  globalThis.seqInput("\x0c"); await settle(); compare("after Ctrl+L (screen change)");
} else {
  for (const k of ["\x1b[C", "\x1b[C", "\x1b[B", "\t"]) { globalThis.seqInput(k); await settle(); }
  compare("after a few keys");
  // An SGR mouse report with the Alt bit (8) reaches the program as Alt: holding
  // Alt over the map labels everything on it, so the frame changes; the same
  // report without it does not. The page's xterm.js does the click-to-SGR step
  // (not modelled by the headless terminal), so the reports are written here.
  const frame = () => globalThis.seqLastView;
  globalThis.seqInput("\x1b[<35;60;12M"); await settle(); // motion, no button
  const plain = frame();
  globalThis.seqInput("\x1b[<35;61;12M"); await settle();
  globalThis.seqInput("\x1b[<43;61;12M"); await settle(); // the same, with Alt
  const alt = frame();
  const count = (f) => strip(f).split("Cleric Trainer").length - 1; // Nearby lists it; Alt adds its label on the map
  const altOk = count(alt) > count(plain);
  console.log(`  ${altOk ? "ok  " : "FAIL"} ${mode}: an SGR report with the Alt bit reaches the program as Alt (labels drawn)`);
  if (!altOk) failed++;
  compare("after an Alt mouse report");
  term.resize(100, 30);
  globalThis.seqTerm.cols = 100; globalThis.seqTerm.rows = 30;
  globalThis.seqResize(100, 30); await settle(); compare("after a resize to 100x30");
}
const ok = expectFail ? failed > 0 : failed === 0;
if (expectFail && !failed) console.log(`  ${mode}: expected the check to fail and it did not`);
process.exit(ok ? 0 : 1);
