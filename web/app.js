// The page for seq-tui as WebAssembly (docs/browser.md). It draws a terminal with
// xterm.js and runs seq-tui.wasm on it. The names the program and the page share
// (seqTerm, seqInput, seqResize, seqExited) are described in cmd/seq-tui-web.
"use strict";

const MIN_COLS = 80, MIN_ROWS = 24;
const $ = (id) => document.getElementById(id);
const isMac = /Mac|iPhone|iPad/.test(navigator.platform);

// intercepted is keys.json's list, the keys of the game that the browser also
// uses (Ctrl+R reloads, Ctrl+L focuses the address bar, F1 opens help, ...).
let intercepted = new Set(["ctrl+a", "ctrl+k", "ctrl+l", "ctrl+r", "ctrl+u", "f1"]);

// keyName names a key event as the game's key table does.
function keyName(e) {
  if (/^F\d{1,2}$/.test(e.key) && !e.ctrlKey && !e.altKey && !e.metaKey) return e.key.toLowerCase();
  if (e.ctrlKey && !e.altKey && !e.metaKey && !e.shiftKey && e.key.length === 1) return "ctrl+" + e.key.toLowerCase();
  return "";
}

// plainReason puts the server's refusal of a character in words.
function plainReason(msg) {
  if (/SELECT_REFUSED|not_permitted/.test(msg)) {
    return "That character isn't on this account; leave Character id empty to pick one.";
  }
  return msg;
}

let running = false;
window.addEventListener("beforeunload", (e) => {
  if (!running) return;
  e.preventDefault();
  e.returnValue = ""; // the game is running: closing the tab leaves it
});

async function start(settings) {
  $("login").style.display = "none";
  $("loading").style.display = "block";
  await document.fonts.load('14px "DejaVu Sans Mono"');
  await document.fonts.load('bold 14px "DejaVu Sans Mono"');

  const term = new Terminal({
    fontFamily: '"DejaVu Sans Mono", monospace',
    fontSize: 15,
    scrollback: 0,
    // The wasm has no TTY, so bubbletea maps "\n" to "\r\n" itself (mapNl) and
    // its renderer assumes a line feed returns to column 0. xterm.js moves only
    // down on "\n" unless convertEol is on, which drifts every later column move.
    convertEol: true,
    cursorBlink: false,
    customGlyphs: true,       // box and block characters drawn by xterm, edge to edge
    macOptionIsMeta: isMac,   // Option is Alt, as the game's Alt keys expect
    altClickMovesCursor: false,
    allowProposedApi: true,
    theme: { background: "#000000", foreground: "#d0d0d0" },
  });
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open($("terminal"));
  try { // customGlyphs needs the WebGL renderer; the DOM one still works without
    const gl = new WebglAddon.WebglAddon();
    gl.onContextLoss(() => gl.dispose());
    term.loadAddon(gl);
  } catch (err) {
    console.warn("no WebGL renderer, drawing with the DOM one:", err);
  }
  fit.fit();
  term.focus();

  term.attachCustomKeyEventHandler((e) => {
    if (e.type === "keydown" && intercepted.has(keyName(e))) e.preventDefault();
    return true;
  });
  term.onData((d) => { if (window.seqInput) window.seqInput(d); });

  const note = () => { $("small").style.display = term.cols < MIN_COLS || term.rows < MIN_ROWS ? "block" : "none"; };
  term.onResize(({ cols, rows }) => { note(); if (window.seqResize) window.seqResize(cols, rows); });
  new ResizeObserver(() => fit.fit()).observe($("terminal"));
  note();

  window.seqTerm = term;
  window.seqExited = (msg) => {
    running = false;
    term.dispose();
    if (msg) {
      sessionStorage.removeItem("seq.character"); // a refused id is not offered again
    } else if (sessionStorage.getItem("seq.character.tried")) {
      sessionStorage.setItem("seq.character", sessionStorage.getItem("seq.character.tried"));
    }
    sessionStorage.removeItem("seq.character.tried");
    $("endedmsg").textContent = msg ? "SEQ stopped: " + plainReason(msg) : "Thanks for playing.";
    $("ended").style.display = "flex";
  };

  const go = new Go();
  go.env = { SEQ_SESSION_TOKEN: settings.token };
  if (settings.character) go.env.SEQ_CHARACTER = settings.character;
  if (settings.newchar) go.env.SEQ_NEW = "1";
  if (settings.url) go.env.SEQ_URL = settings.url;
  let inst;
  try {
    inst = await instantiate(go);
  } catch (err) {
    window.seqExited("the program did not load: " + err);
    return;
  }
  $("loading").style.display = "none";
  running = true;
  go.run(inst);
}

async function instantiate(go) {
  const url = "seq-tui.wasm";
  if (WebAssembly.instantiateStreaming) {
    try { return (await WebAssembly.instantiateStreaming(fetch(url), go.importObject)).instance; }
    catch (err) { console.warn("streaming load failed, trying a plain one:", err); }
  }
  const r = await fetch(url);
  if (!r.ok) throw new Error(url + ": " + r.status);
  return (await WebAssembly.instantiate(await r.arrayBuffer(), go.importObject)).instance;
}

$("again").addEventListener("click", () => location.reload());
$("token").value = sessionStorage.getItem("seq.token") || "";
// Only an id that worked is offered again (seq.character is set when a run ends
// without an error); the field is empty otherwise.
$("character").value = sessionStorage.getItem("seq.character") || "";
$("loginform").addEventListener("submit", async (e) => {
  e.preventDefault();
  const settings = {
    token: $("token").value.trim(),
    character: $("character").value.trim(),
    newchar: $("newchar").checked,
    url: new URLSearchParams(location.search).get("url") || "", // development: another server
  };
  sessionStorage.setItem("seq.token", settings.token);
  sessionStorage.removeItem("seq.character");
  if (settings.character) sessionStorage.setItem("seq.character.tried", settings.character);
  else sessionStorage.removeItem("seq.character.tried");
  start(settings);
});

fetch("keys.json").then((r) => r.json()).then((k) => { intercepted = new Set(k.intercept); }).catch(() => {});
