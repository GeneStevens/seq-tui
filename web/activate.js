// /activate: the page the terminal's login sends the player to. It shows which
// account is approving which code, and approves or denies it. The code comes
// prefilled from ?code=ABCD-EFGH (verification_uri_complete).
"use strict";
(async function () {
  const $ = (id) => document.getElementById(id);
  const auth = SeqAuth.createAuth({ fetch: window.fetch.bind(window), local: localStorage, session: sessionStorage, stytchLib: window.stytch, origin: location.origin });
  const show = (v) => { $("view").innerHTML = SeqSignInUI.renderActivate(v); const f = $("view").querySelector("[autofocus]"); if (f) f.focus(); };
  const cfg = await auth.config();
  if (!cfg) { show({ screen: "error", error: "Browser sign-in is not turned on for this server." }); return; }
  let code = (new URLSearchParams(location.search).get("code") || "").trim();

  const load = async () => {
    const v = await SeqAuth.activateLoad(auth, code);
    show(v);
    $("signin").hidden = v.screen !== "signin";
    if (v.screen === "signin") signIn();
  };
  const signIn = () => {
    if (signIn.done) return;
    signIn.done = true;
    // Google and emailed links leave the page and come back through /auth/callback, which returns here.
    const flow = SeqAuth.createFlow(auth, cfg, { returnTo: "/activate?code=" + encodeURIComponent(code), onSignedIn: load });
    const draw = SeqSignInUI.mount($("signin"), flow);
    flow.onView = draw;
    draw(flow.view);
  };

  $("view").addEventListener("click", async (e) => {
    const b = e.target.closest("[data-act]");
    if (!b) return;
    show({ screen: "loading" });
    const v = await SeqAuth.activateDecide(auth, code, b.dataset.act === "approve" ? "confirm" : "deny");
    if (v.screen === "signin") return load(); // the sign-in ran out while the page was open
    show(v);
  });
  $("view").addEventListener("submit", (e) => {
    e.preventDefault();
    code = new FormData(e.target).get("code").trim();
    load();
  });
  load();
})();
