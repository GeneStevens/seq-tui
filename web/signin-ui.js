// The sign-in panel's drawing (D119): render() turns a view of auth.js's flow
// into HTML, and mount() puts it in an element and sends the player's clicks
// back to the flow. render is pure, so scripts/web-signin-check.mjs tests it.
"use strict";
(function (root) {
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const err = (v) => (v.error ? '<p class="error" role="alert">' + esc(v.error) + "</p>" : "");

  function render(v) {
    switch (v.screen) {
      case "busy":
        return '<p class="busy">' + esc(v.text) + "</p>";
      case "sent":
        return '<p>We sent a sign-in link to <b>' + esc(v.email) + "</b>. Open it on this device to sign in.</p>" +
          '<p class="note">If the email does not come, check your spam folder.</p>' +
          (v.emailOtp ? '<p><button type="button" data-act="code">Use a code instead</button> ' : "<p>") +
          '<button type="button" class="link" data-act="back">Back</button></p>';
      case "code":
        return '<form data-submit="verify"><p>We emailed a code to <b>' + esc(v.email) + "</b>.</p>" + err(v) +
          '<label for="signin-code">Code</label><input id="signin-code" name="code" type="text" inputmode="numeric" autocomplete="one-time-code" spellcheck="false" autofocus>' +
          '<button type="submit">Sign in</button> <button type="button" class="link" data-act="back">Back</button></form>';
      case "newaccount":
        return '<form data-submit="linkAccount"><p>Signed in. This is a new SEQ account (<b>' + esc(v.account) + "</b>).</p>" + err(v) +
          '<p><b>Already have a SEQ account?</b> Link it: enter an old invite code or session token once.</p>' +
          '<input name="credential" type="password" autocomplete="off" spellcheck="false" placeholder="invite code or seq_st_...">' +
          '<button type="submit">Link my account</button> <button type="button" data-act="skipLink">No, start new</button></form>';
      case "done":
        return '<p class="busy">Signed in' + (v.account ? " as <b>" + esc(v.account) + "</b>" : "") + ".</p>";
      default: // choose
        return '<form data-submit="link">' + err(v) +
          (v.google ? '<button type="button" class="google" data-act="google">Sign in with Google</button><p class="or">or</p>' : "") +
          '<label for="signin-email">Email</label><input id="signin-email" name="email" type="email" autocomplete="email" spellcheck="false" placeholder="you@example.com" value="' + esc(v.email) + '">' +
          '<button type="submit">Email me a sign-in link</button>' +
          (v.emailOtp ? ' <button type="button" class="link" data-act="code">Use a code instead</button>' : "") + "</form>";
    }
  }

  // renderActivate draws a view of the /activate page.
  function renderActivate(v) {
    switch (v.screen) {
      case "needcode":
        return '<form data-submit="code"><p>Enter the code your terminal shows.</p>' +
          '<input name="code" type="text" autocomplete="off" spellcheck="false" placeholder="ABCD-EFGH" value="' + esc(v.code) + '" autofocus>' +
          '<button type="submit">Continue</button></form>';
      case "confirm":
        return "<p>Approve sign-in for <b>" + esc(v.email || v.account) + "</b> on seq-tui?</p>" +
          '<p>Code <span class="code">' + esc(v.code) + "</span></p>" +
          "<p>Signing in as <b>" + esc(v.email || v.account) + "</b></p>" +
          '<p class="note">Only approve a code you see in your own terminal right now.</p>' +
          '<button type="button" data-act="approve">Approve</button> <button type="button" class="link" data-act="deny">Deny</button>';
      case "approved":
        return "<p><b>Approved.</b> Go back to your terminal: it signs in by itself. You can close this page.</p>";
      case "denied":
        return "<p>Denied. The terminal was not signed in.</p>";
      case "signin":
        return '<p>Sign in to approve the code <span class="code">' + esc(v.code) + "</span>.</p>";
      case "error":
        return '<p class="error" role="alert">' + esc(v.error) + "</p>";
      default:
        return "<p>Loading…</p>";
    }
  }

  // mount draws the flow's views in el and sends clicks and submits to flow.act.
  function mount(el, flow) {
    const data = () => {
      const d = {};
      el.querySelectorAll("input").forEach((i) => { d[i.name] = i.value; });
      return d;
    };
    const draw = (v) => { el.innerHTML = render(v); const f = el.querySelector("[autofocus]"); if (f) f.focus(); };
    el.addEventListener("click", (e) => {
      const b = e.target.closest("[data-act]");
      if (b) { e.preventDefault(); flow.act(b.dataset.act, data()); }
    });
    el.addEventListener("submit", (e) => {
      const f = e.target.closest("[data-submit]");
      if (f) { e.preventDefault(); flow.act(f.dataset.submit, data()); }
    });
    return draw;
  }

  const api = { render, renderActivate, mount, esc };
  if (typeof module === "object" && module.exports) module.exports = api;
  else root.SeqSignInUI = api;
})(typeof globalThis !== "undefined" ? globalThis : this);
