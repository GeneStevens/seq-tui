// The browser sign-in check (D119): web/auth.js and web/signin-ui.js run in Node
// against stubs for the server (/auth/config, /auth/stytch, /auth/link,
// /auth/device/approve), Stytch's SDK and the browser's storage. It covers the
// panel's flow (Google, an emailed link, a code, linking an old account, sign
// out), a saved session and its expiry, the /auth/callback page (a link that
// works, one that fails and offers a code), and the /activate page (approve,
// deny, not signed in), and holds the pages' markup to what they must load.
// Usage: node scripts/web-signin-check.mjs
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(import.meta.url);
const SeqAuth = require(path.join(root, "web", "auth.js"));
const UI = require(path.join(root, "web", "signin-ui.js"));

let bad = 0, n = 0;
const ok = (cond, what) => { n++; if (!cond) { bad++; console.log("  FAIL " + what); } };
const eq = (got, want, what) => ok(JSON.stringify(got) === JSON.stringify(want), `${what}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);

const store = () => { const m = new Map(); return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)), removeItem: (k) => m.delete(k), m }; };
const CFG = { stytch_project_id: "project-test-1", stytch_public_token: "public-token-test-1", stytch_domain: "https://x.customers.stytch.dev", google: true, email_otp: true };
const SESSION = { token: "seq_st_new", account: "p-abc", expires_ms: 4102444800000, created: true, has_characters: false };

// world builds the stubs for one case. server answers by path; stytch records its calls.
function world(over = {}) {
  const w = { calls: [], posts: [], local: store(), session: store(), clock: 1000, cfg: CFG, stytchFail: null, jwt: "jwt-1", signedIn: true, ...over };
  w.fetch = async (url, init) => {
    const body = init && init.body ? JSON.parse(init.body) : null;
    if (init && init.method === "POST") w.posts.push([url, body]);
    const reply = (status, data) => ({ ok: status === 200, status, json: async () => data, headers: { get: () => "application/json" } });
    if (url === "/auth/callback/") return w.callbackServed ? { ok: true, status: 200, headers: { get: () => "text/html; charset=utf-8" } } : { ok: false, status: 404, headers: { get: () => "text/plain" } };
    if (url === "/auth/config") return w.cfg ? reply(200, w.cfg) : reply(404, {});
    if (url === "/auth/stytch") return body.session === "bad" ? reply(401, { error: "invalid_session", message: "That sign-in has expired." }) : reply(200, w.sessionReply || SESSION);
    if (url === "/auth/link") return body.credential === "wrong" ? reply(403, { error: "invalid_credential", message: "That invite code or session token is not valid." }) : reply(200, { token: "seq_st_old", account: "old-acct", expires_ms: 4102444800000 });
    if (url === "/auth/device/approve") {
      if (body.user_code === "ZZZZ-ZZZZ") return reply(404, { error: "invalid_code", message: "That code is wrong, has expired or was already used." });
      return reply(200, { account: "p-abc", email: "a@b.c", status: body.deny ? "denied" : body.confirm ? "approved" : "confirm" });
    }
    return reply(404, {});
  };
  const call = (name, ...a) => { w.calls.push([name, ...a]); if (w.stytchFail === name) throw new Error("stytch said no"); };
  w.stytchLib = { StytchHeadlessClient: class {
    constructor(token) { w.calls.push(["client", token]); }
    get oauth() { return { google: { start: async (o) => call("google.start", o) }, authenticate: async (t, o) => { call("oauth.authenticate", t); return { session_jwt: w.jwt }; } }; }
    get magicLinks() { return { email: { loginOrCreate: async (e, o) => call("link.send", e, o) }, authenticate: async (t, o) => { call("link.authenticate", t); return { session_jwt: w.jwt }; } }; }
    get otps() { return { email: { loginOrCreate: async (e, o) => { call("otp.send", e, o); return { method_id: "mid-1" }; } }, authenticate: async (c, m, o) => { call("otp.authenticate", c, m); if (c === "000000") throw new Error("That code is not right."); return { session_jwt: w.jwt }; } }; }
    get session() { return { getSync: () => (w.signedIn ? { id: "s" } : null), authenticate: async () => ({ session_jwt: w.jwt }), revoke: async () => call("session.revoke") }; }
  } };
  w.auth = () => SeqAuth.createAuth({ fetch: w.fetch, local: w.local, session: w.session, stytchLib: w.stytchLib, origin: "https://localhost:8443", now: () => w.clock });
  return w;
}
const calls = (w, name) => w.calls.filter((c) => c[0] === name);

async function flowCase(name, fn) { console.log("  " + name); await fn(); }

// ---- the server has no sign-in: the page offers the token form only
await flowCase("sign-in off", async () => {
  ok((await world({ cfg: null }).auth().config()) === null, "no /auth/config: config is null");
  ok((await world({ cfg: { ...CFG, stytch_project_id: "" } }).auth().config()) === null, "empty project: config is null");
});

// ---- Google, an emailed link, then a code instead
await flowCase("google, link, code, new account, link to an old one", async () => {
  const w = world();
  const auth = w.auth();
  eq((await auth.config()).stytch_public_token, "public-token-test-1", "config");
  const seen = [], signed = [];
  const flow = SeqAuth.createFlow(auth, CFG, { offerLink: true, returnTo: "/", onSignedIn: (s) => signed.push(s), onView: (v) => seen.push(v.screen) });
  eq(flow.view.screen, "choose", "first view");
  await flow.act("google");
  eq(calls(w, "google.start")[0][1], { login_redirect_url: "https://localhost:8443/", signup_redirect_url: "https://localhost:8443/" }, "behind seqd (no /auth/callback page) google comes back to the root page");
  eq(w.session.getItem(SeqAuth.AFTER_KEY), "/", "return path kept for the callback");
  await flow.act("link", { email: "nope" });
  ok(/email address/.test(flow.view.error), "a bad address is refused in words");
  await flow.act("link", { email: " a@b.c " });
  eq(flow.view.screen, "sent", "link sent");
  const sent = calls(w, "link.send")[0];
  eq([sent[1], sent[2].login_magic_link_url, sent[2].signup_magic_link_url], ["a@b.c", "https://localhost:8443/", "https://localhost:8443/"], "link send args");
  ok(!JSON.stringify(sent[2]).includes("pkce") && !JSON.stringify(sent[2]).includes("code_challenge"), "no PKCE for magic links");
  ok(/Use a code instead/.test(UI.render(flow.view)), "the sent view offers a code");
  await flow.act("code", {});
  eq(flow.view.screen, "code", "code view");
  await flow.act("verify", { code: "000000" });
  eq(flow.view.screen, "code", "a wrong code stays on the code view");
  ok(/not right/.test(flow.view.error), "wrong code message");
  eq(signed.length, 0, "not signed in by a wrong code");
  await flow.act("verify", { code: "123 456" });
  eq(calls(w, "otp.authenticate")[1].slice(1), ["123456", "mid-1"], "spaces dropped from the code");
  eq(w.posts.find((p) => p[0] === "/auth/stytch")[1], { session: "jwt-1" }, "the Stytch session is posted");
  eq(flow.view.screen, "newaccount", "a brand-new account is offered the link");
  eq(signed.length, 0, "not started yet: the offer comes first");
  ok(/Already have a SEQ account\?/.test(UI.render(flow.view)), "offer text");
  eq(JSON.parse(w.local.getItem(SeqAuth.SESSION_KEY)).token, "seq_st_new", "session kept for return visits");
  await flow.act("linkAccount", { credential: "wrong" });
  eq(flow.view.screen, "newaccount", "a refused link stays on the offer");
  ok(/not valid/.test(flow.view.error), "refused link message");
  await flow.act("linkAccount", { credential: "ABCD-1234" });
  eq(w.posts.find((p) => p[0] === "/auth/link" && p[1].credential === "ABCD-1234")[1], { session: "jwt-1", credential: "ABCD-1234" }, "link posts the Stytch session and the old credential");
  eq([signed.length, signed[0].token, signed[0].account], [1, "seq_st_old", "old-acct"], "signed in as the old account");
  eq(JSON.parse(w.local.getItem(SeqAuth.SESSION_KEY)).token, "seq_st_old", "the linked session replaces the new one");
});

await flowCase("the callback page is used when the host serves it", async () => {
  const w = world({ callbackServed: true });
  const auth = w.auth(); await auth.config();
  await auth.emailLink("a@b.c", "/activate?code=ABCD-EFGH");
  const o = calls(w, "link.send")[0][2];
  eq([o.login_magic_link_url, o.signup_magic_link_url], ["https://localhost:8443/auth/callback", "https://localhost:8443/auth/callback"], "link points at /auth/callback");
  eq(w.session.getItem(SeqAuth.AFTER_KEY), "/activate?code=ABCD-EFGH", "return path kept");
});

await flowCase("an existing player skips the offer; skip starts a new account", async () => {
  const signed = [];
  const w2 = world(); const auth2 = w2.auth(); await auth2.config();
  const f2 = SeqAuth.createFlow(auth2, CFG, { offerLink: true, onSignedIn: (s) => signed.push(s) });
  await f2.act("code", { email: "a@b.c" }); await f2.act("verify", { code: "111111" });
  eq(f2.view.screen, "newaccount", "new account offered");
  await f2.act("skipLink");
  eq(signed.length, 1, "skip signs in");
  const w3 = world({ sessionReply: { ...SESSION, created: false, has_characters: true } });
  const a3 = w3.auth(); await a3.config();
  const f3 = SeqAuth.createFlow(a3, CFG, { offerLink: true, onSignedIn: (s) => signed.push(s) });
  await f3.act("code", { email: "a@b.c" }); await f3.act("verify", { code: "111111" });
  eq([f3.view.screen, signed.length], ["done", 2], "a known player goes straight in");
});

await flowCase("stytch errors come back as words on the first view", async () => {
  const w = world({ stytchFail: "link.send" });
  const auth = w.auth(); await auth.config();
  const flow = SeqAuth.createFlow(auth, CFG, { onSignedIn() {} });
  await flow.act("link", { email: "a@b.c" });
  eq([flow.view.screen, flow.view.error], ["choose", "stytch said no"], "send failure");
  ok(/stytch said no/.test(UI.render(flow.view)), "rendered");
});

// ---- saved sessions and sign-out
await flowCase("saved session, expiry, sign out", async () => {
  const w = world();
  const auth = w.auth(); await auth.config();
  eq(auth.saved(), null, "nothing saved");
  await auth.exchange("jwt-1");
  eq(auth.saved().account, "p-abc", "saved after sign-in");
  w.clock = 4102444800001;
  eq(auth.saved(), null, "expired session is not offered");
  eq(w.local.getItem(SeqAuth.SESSION_KEY), null, "and is dropped");
  w.clock = 1000;
  await auth.exchange("jwt-1");
  await auth.signOut();
  eq([auth.saved(), calls(w, "session.revoke").length], [null, 1], "sign out forgets and revokes");
  let msg = ""; try { await auth.exchange("bad"); } catch (e) { msg = e.message; }
  ok(/expired/.test(msg), "a refused exchange says why");
});

// ---- /auth/callback
await flowCase("callback: link, google, failures", async () => {
  let w = world(); let auth = w.auth(); await auth.config();
  w.session.setItem(SeqAuth.AFTER_KEY, "/activate?code=ABCD-EFGH");
  let r = await SeqAuth.callbackPage(auth, "?stytch_token_type=magic_links&token=tok1");
  eq([r.ok, r.to, calls(w, "link.authenticate")[0][1]], [true, "/activate?code=ABCD-EFGH", "tok1"], "magic link completes and returns");
  eq(w.session.getItem(SeqAuth.AFTER_KEY), null, "return path used once");
  r = await SeqAuth.callbackPage(auth, "?stytch_token_type=oauth&token=tok2");
  eq([r.ok, r.to, calls(w, "oauth.authenticate")[0][1]], [true, "/", "tok2"], "google completes");
  w = world({ stytchFail: "link.authenticate" }); auth = w.auth(); await auth.config();
  r = await SeqAuth.callbackPage(auth, "?stytch_token_type=magic_links&token=tok1");
  ok(!r.ok && r.hintCode && /use a code instead/i.test(r.error), "a link that fails on another device offers a code");
  w = world({ stytchFail: "oauth.authenticate" }); auth = w.auth(); await auth.config();
  r = await SeqAuth.callbackPage(auth, "?stytch_token_type=oauth&token=tok2");
  ok(!r.ok && !r.hintCode, "a google failure has no code hint");
  r = await SeqAuth.callbackPage(auth, "");
  ok(!r.ok, "a callback with no token says so");
});

// ---- /activate
await flowCase("activate: not signed in, approve, deny, bad code", async () => {
  let w = world({ signedIn: false }); let auth = w.auth(); await auth.config();
  eq((await SeqAuth.activateLoad(auth, "")).screen, "needcode", "no code asks for it");
  eq((await SeqAuth.activateLoad(auth, "ABCD-EFGH")).screen, "signin", "not signed in asks to sign in");
  w = world(); auth = w.auth(); await auth.config();
  let v = await SeqAuth.activateLoad(auth, "abcd-efgh");
  eq([v.screen, v.code, v.account], ["confirm", "ABCD-EFGH", "p-abc"], "signed in sees what is approved");
  const html = UI.renderActivate(v);
  ok(/Approve sign-in for <b>seq-tui<\/b> on this device\?/.test(html) && /ABCD-EFGH/.test(html) && /p-abc/.test(html), "confirm text names the app, code and account");
  eq(w.posts.find((p) => p[0] === "/auth/device/approve")[1], { session: "jwt-1", user_code: "abcd-efgh", confirm: false, deny: false }, "the check does not approve");
  eq((await SeqAuth.activateDecide(auth, "ABCD-EFGH", "confirm")).screen, "approved", "approve");
  eq((await SeqAuth.activateDecide(auth, "ABCD-EFGH", "deny")).screen, "denied", "deny");
  v = await SeqAuth.activateLoad(auth, "ZZZZ-ZZZZ");
  ok(v.screen === "error" && /wrong/.test(v.error), "a wrong code says so");
});

// ---- drawing
await flowCase("render", async () => {
  ok(!/Google/.test(UI.render({ screen: "choose", google: false, emailOtp: true, email: "" })), "no Google button when off");
  ok(!/Use a code instead/.test(UI.render({ screen: "choose", google: true, emailOtp: false, email: "" })), "no code option when off");
  const h = UI.render({ screen: "choose", google: true, emailOtp: true, email: '"><script>x</script>' });
  ok(/Sign in with Google/.test(h) && /Email me a sign-in link/.test(h) && /Use a code instead/.test(h), "choose view");
  ok(!/<script>x/.test(h), "input is escaped");
});

// ---- the pages' markup
await flowCase("pages", async () => {
  const read = (...p) => readFileSync(path.join(root, "web", ...p), "utf8");
  const index = read("index.html");
  ok(/Other ways to sign in/.test(index) && /id="loginform"[^>]*hidden/.test(index), "index: the token form is behind Other ways to sign in");
  for (const p of ["index.html", "activate/index.html", "auth/callback/index.html"]) {
    const h = read(p);
    ok(/vendor\/stytch-vanilla-js-6\.3\.0\.headless\.js/.test(h) && /auth\.js/.test(h) && /signin-ui\.js/.test(h), p + " loads the pinned SDK and the sign-in code");
  }
  ok(!/js\.stytch\.com/.test(index), "the SDK is vendored, not loaded from a CDN");
  ok(/approve-?|activate\.js/.test(read("activate", "index.html")), "activate page script");
});

console.log(bad ? `web-signin-check: ${bad} of ${n} checks FAILED` : `web-signin-check: ${n} checks ok`);
process.exit(bad ? 1 : 0);
