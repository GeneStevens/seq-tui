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

// keepShiftSelecting makes Shift-click select text on every platform. xterm.js
// forces a selection with Shift only off macOS (on a Mac it is Option, which the
// game needs for walking), so a Mac's Shift-click would go to the game as a mouse
// report and copying would be lost. It uses an internal, so it is guarded: if the
// internal is gone, Shift-click behaves as the xterm.js build does.
function keepShiftSelecting(term) {
  try {
    const sel = term._core._selectionService;
    sel.shouldForceSelection = (e) => e.shiftKey;
  } catch (err) {
    console.warn("Shift-click may not select text on this xterm.js:", err);
  }
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
    // Option-click is the game's walking click (the terminal's too): it must reach
    // the program, with the Alt bit in the mouse report, and not select text.
    macOptionClickForcesSelection: false,
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
  keepShiftSelecting(term);

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
    if (settings.fromSaved && /hello refused/.test(msg || "")) { // the saved sign-in ran out or was revoked
      localStorage.removeItem(SeqAuth.SESSION_KEY);
      msg = "";
      $("endedmsg").textContent = "Your saved sign-in has expired. Sign in again.";
      $("ended").style.display = "flex";
      return;
    }
    if (msg) {
      sessionStorage.removeItem("seq.character"); // a refused id is not offered again
    } else if (sessionStorage.getItem("seq.character.tried")) {
      sessionStorage.setItem("seq.character", sessionStorage.getItem("seq.character.tried"));
    }
    sessionStorage.removeItem("seq.character.tried");
    $("endedmsg").textContent = msg ? "SEQ stopped: " + plainReason(msg) : "Thanks for playing.";
    $("endsignout").hidden = !auth.saved();
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
$("endsignout").addEventListener("click", async () => { await auth.signOut(); location.reload(); });
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

const auth = SeqAuth.createAuth({ fetch: window.fetch.bind(window), local: localStorage, session: sessionStorage, stytchLib: window.stytch, origin: location.origin });
const pageURL = () => new URLSearchParams(location.search).get("url") || "";

// startSignedIn plays with the seqd session a sign-in gave (the picker lists the
// account's characters, so no character id is asked for).
function startSignedIn(s, fromSaved) {
  history.replaceState(null, "", location.pathname + (pageURL() ? "?url=" + encodeURIComponent(pageURL()) : ""));
  start({ token: s.token, character: "", newchar: false, url: pageURL(), fromSaved });
}

// boot decides the first screen: the token form alone when the server has no
// sign-in; a sign-in coming back to this page (a link or Google that redirected
// to the root) is finished; a saved session gets a welcome and one click to play;
// otherwise the sign-in panel, with the token form behind "Other ways to sign in".
async function boot() {
  const cfg = await auth.config();
  if (!cfg) { $("loginform").hidden = false; return; }
  $("other").addEventListener("click", (e) => { e.preventDefault(); $("loginform").hidden = !$("loginform").hidden; });
  $("otherlink").hidden = false;
  const flow = SeqAuth.createFlow(auth, cfg, { offerLink: true, returnTo: "/", onSignedIn: (s) => startSignedIn(s, false) });
  const draw = SeqSignInUI.mount($("signin"), flow);
  flow.onView = draw;
  const back = () => { $("welcome").hidden = true; $("signin").hidden = false; draw(flow.view); };
  if (new URLSearchParams(location.search).has("stytch_token_type")) {
    $("signin").hidden = false;
    try {
      const s = await auth.completeCallback(location.search);
      if (s && s.returnTo && s.returnTo !== "/") return location.replace(s.returnTo); // e.g. /activate?code=…
      if (s) return startSignedIn(s, false);
    }
    catch (e) { history.replaceState(null, "", location.pathname); flow.view = { screen: "choose", google: !!cfg.google, emailOtp: cfg.email_otp !== false, error: e.message }; draw(flow.view); return; }
  }
  const s = auth.saved();
  if (s) {
    $("who").textContent = SeqAuth.who(s);
    $("welcome").hidden = false;
    $("play").addEventListener("click", () => startSignedIn(s, true));
    $("signout").addEventListener("click", async () => { await auth.signOut(); $("welcome").hidden = true; $("signin").hidden = false; draw(flow.view); });
    $("play").focus();
    return;
  }
  back();
}
boot();

fetch("keys.json").then((r) => r.json()).then((k) => { intercepted = new Set(k.intercept); }).catch(() => {});
