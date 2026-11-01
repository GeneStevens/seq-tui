// Player sign-in for the pages (D119): Google, an emailed sign-in link or an
// emailed code, through Stytch's headless JavaScript SDK (vendor/stytch-*.js),
// then the exchange of the Stytch session for seqd's own session token at
// POST /auth/stytch. Everything here takes its browser parts (fetch, storage,
// the SDK, the page's origin) as arguments, so scripts/web-signin-check.mjs runs
// it in Node against stubs. The DOM is in signin-ui.js.
"use strict";
(function (root) {
  const SESSION_KEY = "seq.session";   // localStorage: {token, account, email, expires_ms}: kept for return visits
  const AFTER_KEY = "seq.after";       // sessionStorage: where the callback goes on (the /activate page and its code)
  const CALLBACK_PATH = "/auth/callback";
  const DEFAULT_SESSION_MINUTES = 60;   // an older seqd does not serve stytch_session_minutes
  const LINK_MINUTES = 30;
  const CODE_MINUTES = 10;

  // plain puts an error in words for the player.
  function plain(err) {
    const m = (err && (err.message || err.error_message || err.error_type)) || String(err || "");
    return m || "Something went wrong. Try again.";
  }

  function createAuth(deps) {
    const { fetch, local, session, stytchLib, origin } = deps;
    const now = deps.now || (() => Date.now());
    let cfg = null, client = null;

    // sessionMinutes is the Stytch session length the server asks for: the
    // sign-in only has to last until POST /auth/stytch trades it for seqd's own.
    function sessionMinutes() {
      const m = cfg && Number(cfg.stytch_session_minutes);
      return m > 0 ? Math.floor(m) : DEFAULT_SESSION_MINUTES;
    }

    async function json(path, body) {
      let r;
      try {
        r = await fetch(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
      } catch (e) {
        throw new Error("Cannot reach the server. Check your connection and try again.");
      }
      let data = {};
      try { data = await r.json(); } catch (e) { /* not JSON */ }
      if (!r.ok) {
        const err = new Error(data.message || "The server refused that (" + r.status + ").");
        err.code = data.error;
        throw err;
      }
      return data;
    }

    // config asks the server for the Stytch settings; null means sign-in is off
    // (no /auth/config, or no project) and the page offers the token form only.
    async function config() {
      try {
        const r = await fetch("/auth/config");
        if (!r.ok) return null;
        const c = await r.json();
        if (!c || !c.stytch_project_id || !c.stytch_public_token) return null;
        cfg = c;
        return c;
      } catch (e) {
        return null;
      }
    }

    function stytch() {
      if (!cfg) throw new Error("Sign-in is not turned on for this server.");
      if (!client) client = new stytchLib.StytchHeadlessClient(cfg.stytch_public_token);
      return client;
    }

    function saved() {
      let s;
      try { s = JSON.parse(local.getItem(SESSION_KEY) || "null"); } catch (e) { s = null; }
      if (!s || !s.token || !(s.expires_ms > now())) {
        if (s) local.removeItem(SESSION_KEY);
        return null;
      }
      return s;
    }

    // exchange trades a Stytch session JWT for seqd's session and keeps it.
    async function exchange(jwt) {
      if (!jwt) throw new Error("Sign-in did not finish: no session came back. Try again.");
      const r = await json("/auth/stytch", { session: jwt });
      const s = { token: r.token, account: r.account, email: r.email || "", expires_ms: r.expires_ms };
      local.setItem(SESSION_KEY, JSON.stringify(s));
      return { ...s, created: !!r.created, hasCharacters: !!r.has_characters };
    }

    // signOut forgets the seqd session and ends the Stytch one.
    async function signOut() {
      local.removeItem(SESSION_KEY);
      try { if (cfg) await stytch().session.revoke({ forceClear: true }); } catch (e) { /* already gone */ }
    }

    // callbackURL is where Stytch sends the player back: /auth/callback. seqd's
    // own listener routes everything under /auth/ to its sign-in endpoints, so
    // behind it that path is a 404 and the page that finishes the sign-in is the
    // root page, which takes the same ?stytch_token_type=…&token=… query. The
    // probe picks the callback page when the host serves it (a proxy in front
    // that routes it to the files), else the root. Both URLs are Redirect URLs of
    // the Stytch project (docs/browser.md).
    let callback = null;
    async function callbackURL() {
      if (callback) return callback;
      callback = origin + "/";
      try {
        const r = await fetch(CALLBACK_PATH + "/", { method: "GET" });
        const type = r.headers && r.headers.get ? r.headers.get("content-type") || "" : "";
        if (r.ok && /html/.test(type)) callback = origin + CALLBACK_PATH;
      } catch (e) { /* the root page */ }
      return callback;
    }

    // rememberReturn keeps where to go after a sign-in that leaves the page (a
    // Google redirect or an emailed link): path and search of this page.
    function rememberReturn(target) { if (target) session.setItem(AFTER_KEY, target); }
    function takeReturn() { const t = session.getItem(AFTER_KEY); session.removeItem(AFTER_KEY); return t; }

    async function google(returnTo) {
      rememberReturn(returnTo);
      const url = await callbackURL();
      return stytch().oauth.google.start({ login_redirect_url: url, signup_redirect_url: url });
    }

    async function emailLink(email, returnTo) {
      rememberReturn(returnTo);
      const url = await callbackURL();
      return stytch().magicLinks.email.loginOrCreate(email, {
        login_magic_link_url: url, signup_magic_link_url: url,
        login_expiration_minutes: LINK_MINUTES, signup_expiration_minutes: LINK_MINUTES,
      });
    }

    async function emailCode(email) {
      const r = await stytch().otps.email.loginOrCreate(email, { expiration_minutes: CODE_MINUTES });
      return r.method_id;
    }

    async function verifyCode(methodId, code) {
      const r = await stytch().otps.authenticate(String(code).replace(/\s+/g, ""), methodId, { session_duration_minutes: sessionMinutes() });
      return exchange(r.session_jwt);
    }

    // completeCallback finishes a sign-in that came back to the page with
    // ?stytch_token_type=…&token=…. A link that does not work (opened too late,
    // twice, or on a device the sign-in cannot continue on) is an error whose
    // hint is the code.
    async function completeCallback(search) {
      const q = new URLSearchParams(search);
      const type = q.get("stytch_token_type"), token = q.get("token");
      if (!type || !token) return null;
      let r;
      try {
        if (type === "magic_links") r = await stytch().magicLinks.authenticate(token, { session_duration_minutes: sessionMinutes() });
        else if (type === "oauth") r = await stytch().oauth.authenticate(token, { session_duration_minutes: sessionMinutes() });
        else throw new Error("Unknown sign-in type.");
      } catch (e) {
        const err = new Error(type === "magic_links"
          ? "That sign-in link did not work: it may have expired, been used already, or been opened on a different device than you asked for it on. Go back and use a code instead."
          : "Google sign-in did not finish: " + plain(e));
        err.hintCode = type === "magic_links";
        throw err;
      }
      const s = await exchange(r.session_jwt);
      return { ...s, returnTo: takeReturn() };
    }

    // stytchJWT is a fresh session JWT of the Stytch sign-in this browser holds
    // (the /activate page proves who is approving with it), or null.
    async function stytchJWT() {
      if (!cfg) return null;
      try {
        const c = stytch();
        if (!c.session.getSync()) return null;
        const r = await c.session.authenticate({ session_duration_minutes: sessionMinutes() });
        return r.session_jwt || null;
      } catch (e) {
        return null;
      }
    }

    // link moves the Stytch sign-in onto an old account, by an invite code or a
    // session token of it.
    async function link(jwt, credential) {
      const r = await json("/auth/link", { session: jwt, credential: credential.trim() });
      const s = { token: r.token, account: r.account, email: r.email || "", expires_ms: r.expires_ms };
      local.setItem(SESSION_KEY, JSON.stringify(s));
      return { ...s, created: false, hasCharacters: true };
    }

    // device asks the server about, approves or denies a terminal's code.
    function device(jwt, userCode, how) {
      return json("/auth/device/approve", { session: jwt, user_code: userCode, confirm: how === "confirm", deny: how === "deny" });
    }

    return { config, saved, exchange, signOut, google, emailLink, emailCode, verifyCode, completeCallback, stytchJWT, link, device, takeReturn, callbackURL };
  }

  // createFlow is the sign-in panel's state machine: act(name, data) does what the
  // player asked and leaves a view for render() to draw. A view is
  // {screen, error?, …}: choose, busy, sent (a link on its way), code (waiting for
  // the digits), newaccount (offer to link) or done.
  function createFlow(auth, cfg, hooks) {
    const flow = { view: { screen: "choose", google: !!cfg.google, emailOtp: cfg.email_otp !== false }, email: "", methodId: "" };
    const set = (v) => { flow.view = v; if (flow.onView) flow.onView(v); };
    const choose = (error) => set({ screen: "choose", google: !!cfg.google, emailOtp: cfg.email_otp !== false, email: flow.email, error });
    const finish = (s) => {
      if (s.created && !s.hasCharacters && hooks.offerLink) set({ screen: "newaccount", account: s.account, session: s });
      else { set({ screen: "done", account: s.account }); hooks.onSignedIn(s); }
    };
    const steps = {
      async google() { set({ screen: "busy", text: "Opening Google…" }); await auth.google(hooks.returnTo); },
      async link({ email }) {
        flow.email = email.trim();
        if (!flow.email.includes("@")) return choose("Enter your email address.");
        set({ screen: "busy", text: "Sending the link…" });
        await auth.emailLink(flow.email, hooks.returnTo);
        set({ screen: "sent", email: flow.email, emailOtp: cfg.email_otp !== false });
      },
      async code({ email }) {
        flow.email = (email || flow.email).trim();
        if (!flow.email.includes("@")) return choose("Enter your email address.");
        set({ screen: "busy", text: "Sending the code…" });
        flow.methodId = await auth.emailCode(flow.email);
        set({ screen: "code", email: flow.email });
      },
      async verify({ code }) {
        if (!String(code || "").trim()) return set({ screen: "code", email: flow.email, error: "Enter the code from the email." });
        set({ screen: "busy", text: "Checking the code…" });
        try {
          finish(await auth.verifyCode(flow.methodId, code));
        } catch (e) {
          set({ screen: "code", email: flow.email, error: plain(e) });
        }
      },
      async linkAccount({ credential }) {
        if (!String(credential || "").trim()) return set({ ...flow.view, error: "Enter an invite code or session token." });
        const prior = flow.view;
        set({ screen: "busy", text: "Linking…" });
        try {
          const jwt = await auth.stytchJWT();
          const s = await auth.link(jwt, credential);
          set({ screen: "done", account: s.account });
          hooks.onSignedIn(s);
        } catch (e) {
          set({ ...prior, error: plain(e) });
        }
      },
      async skipLink() { set({ screen: "done", account: flow.view.account }); hooks.onSignedIn(flow.view.session); },
      async back() { choose(); },
    };
    flow.act = async (name, data) => {
      try {
        await steps[name](data || {});
      } catch (e) {
        if (name === "verify" || name === "linkAccount") return;
        choose(plain(e));
      }
    };
    return flow;
  }

  // callbackPage finishes the sign-in on /auth/callback and says where to go on:
  // {ok: true, to} or {ok: false, error, hintCode} (hintCode: a link that did not
  // work, so the page offers a code instead).
  async function callbackPage(auth, search) {
    try {
      const s = await auth.completeCallback(search);
      if (!s) return { ok: false, error: "This page is where a sign-in link or Google brings you back. Start from the sign-in page.", hintCode: false };
      return { ok: true, to: s.returnTo || "/", session: s };
    } catch (e) {
      return { ok: false, error: plain(e), hintCode: !!e.hintCode };
    }
  }

  // The /activate page, as views: signin (not signed in yet), confirm (who is
  // approving which code), approved, denied, or error.
  const CODE_RE = /^[A-Za-z]{4}-?[A-Za-z]{4}$/;
  async function activateLoad(auth, userCode) {
    if (!CODE_RE.test((userCode || "").trim())) return { screen: "needcode", code: userCode || "" };
    const jwt = await auth.stytchJWT();
    if (!jwt) return { screen: "signin", code: userCode };
    try {
      const r = await auth.device(jwt, userCode, "check");
      return { screen: "confirm", code: userCode.toUpperCase(), account: r.account, email: r.email || "" };
    } catch (e) {
      return { screen: "error", error: plain(e) };
    }
  }
  async function activateDecide(auth, userCode, how) {
    try {
      const jwt = await auth.stytchJWT();
      if (!jwt) return { screen: "signin", code: userCode };
      const r = await auth.device(jwt, userCode, how);
      return { screen: r.status === "denied" ? "denied" : "approved", account: r.account };
    } catch (e) {
      return { screen: "error", error: plain(e) };
    }
  }

  // who is what the page calls a signed-in player: the confirmed email, else the
  // account id (an old saved record has no email).
  function who(s) { return (s && (s.email || s.account)) || ""; }

  const api = { who, createAuth, createFlow, callbackPage, activateLoad, activateDecide, plain, SESSION_KEY, AFTER_KEY, CALLBACK_PATH };
  if (typeof module === "object" && module.exports) module.exports = api;
  else root.SeqAuth = api;
})(typeof globalThis !== "undefined" ? globalThis : this);
