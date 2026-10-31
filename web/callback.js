// /auth/callback: where Stytch brings a player back from Google or an emailed
// link (the URL is a Redirect URL of the Stytch project). The sign-in finishes
// here, then the page goes on to where the player was (the game, or /activate).
"use strict";
(async function () {
  const $ = (id) => document.getElementById(id);
  const auth = SeqAuth.createAuth({ fetch: window.fetch.bind(window), local: localStorage, session: sessionStorage, stytchLib: window.stytch, origin: location.origin });
  const cfg = await auth.config();
  if (!cfg) { $("status").textContent = "Sign-in is not turned on for this server."; $("home").hidden = false; return; }
  const r = await SeqAuth.callbackPage(auth, location.search);
  if (r.ok) { location.replace(r.to); return; }
  const status = $("status");
  status.className = "error";
  status.textContent = r.error;
  $("home").hidden = false;
  if (r.hintCode) { // offer the code right here
    const flow = SeqAuth.createFlow(auth, cfg, { returnTo: "/", onSignedIn: () => location.replace(auth.takeReturn() || "/") });
    $("signin").hidden = false;
    const draw = SeqSignInUI.mount($("signin"), flow);
    flow.onView = draw;
    draw({ screen: "choose", google: false, emailOtp: cfg.email_otp !== false, email: "" });
  }
})();
