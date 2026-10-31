# seq-tui in a browser (seq-design D106)

The same program, compiled to WebAssembly (`GOOS=js GOARCH=wasm`) and drawn by
[xterm.js](https://xtermjs.org/) in a page. It talks to seqd over the same
WebSocket the terminal client uses.

```
make web          # web/dist: the page, seq-tui.wasm, and a .gz and .br of each file
make web-smoke    # builds the wasm and runs it under Node (part of make verify)
```

Both need Go and Node. `make web` prints each file's size. At the time of writing
(xterm.js 6.0.0, DejaVu Sans Mono 2.37):

| file | raw | gzip | brotli |
| --- | --- | --- | --- |
| seq-tui.wasm | 17.2 MB | 4.1 MB | **2.9 MB** |
| xterm.js, WebGL and fit add-ons | 0.7 MB | 0.2 MB | 0.15 MB |
| DejaVu Sans Mono, regular and bold | 0.64 MB | 0.38 MB | 0.32 MB |
| **first visit** | | ~4.7 MB | **~3.4 MB** |

## What the page does

- Signs the player in (D119; see "Sign-in" below): Google, an emailed sign-in
  link, or an emailed code. The seqd session it gets is kept in `localStorage`
  (90 days), so a returning player sees "Welcome back" and one Play button.
- When the server has sign-in off, or under "Other ways to sign in", asks for a
  session token (the one `seq-tui` takes in `SEQ_SESSION_TOKEN`), and
  optionally a character id, or to make a new character. That token is kept in
  the tab's `sessionStorage`, so it survives a reload and goes when the tab
  closes.
- Connects to `wss://<the page's host>/seq/v1`. Open the page as
  `…/?url=wss://other/seq/v1` to name another server (development: the server
  checks the browser's `Origin`, so another host is refused unless it is the page's).
- Keeps Ctrl+A, Ctrl+K, Ctrl+L, Ctrl+R, Ctrl+U and F1 from the browser
  (`preventDefault`), so they reach the game. The list is `web/keys.json`;
  `go test ./web` holds it to the help's binding table, so a new Ctrl+letter or
  function-key binding that is not listed fails. Ctrl+W, Ctrl+T and Ctrl+N cannot
  be taken from a browser: no binding may use them (the same test).
- Asks before the tab is closed or reloaded while the game runs (`beforeunload`).
- On macOS, Option is Alt (`macOptionIsMeta`), and a click does not move xterm's
  cursor. **Option-click walks** (D109) and a plain click never does: xterm.js is
  told `macOptionClickForcesSelection: false` so that Option-click reaches the
  game, as a mouse report with the Alt bit, and does not select text. xterm.js
  forces a selection with Shift only off macOS (on a Mac that is Option), so the
  page makes Shift force it on every platform (`keepShiftSelecting` in
  `web/app.js`, which sets xterm.js's internal `shouldForceSelection`; if that
  internal is ever gone the page logs a warning and a Mac's Shift-click goes to
  the game, whose fallback is then to copy with the browser's own tools).
  Without Option, click then **W**.
- Says so under 80 × 24 (the program shows its own too small note as well).
- Draws box and block characters with xterm's `customGlyphs` (the WebGL renderer;
  without WebGL the DOM renderer is used and they come from the font), in a bundled
  DejaVu Sans Mono, so that `⚠`, `✉` and the map glyphs are one cell wide.

Not in the browser build: duo mode (`-duo`), session logs (it runs as
`-no-record`: the sessions view lists the server's own log only) and
`-replay`. Key releases do not exist in xterm.js, so the walk uses the same
repeat-idle fallback as a terminal without the kitty keyboard protocol
(xterm.js never answers the query, so the program does not expect releases).

## Sign-in (D119)

The pages are `index.html` (the game), `/activate` (approve `seq-tui login`) and
`/auth/callback` (where Stytch brings a player back). They share `auth.js` (the
flow, with the browser parts passed in so `make web-check` can run it against
stubs), `signin-ui.js` (the panel) and Stytch's headless JavaScript SDK.

- **Config.** The page reads `GET /auth/config` from seqd: the Stytch project's
  public token and which methods are on. A 404, or no project, means sign-in is
  off and the page shows the token form alone, as before.
- **Methods.** Google (Stytch OAuth); email: a sign-in **link** by default
  (`magicLinks.email.loginOrCreate`), with **Use a code instead** one click away
  (`otps.email`). No passwords. The Stytch session JWT is posted to
  `POST /auth/stytch`; the answer is a seqd session token, which the game's hello
  takes as it takes any other. A brand-new account (no characters) is offered
  "Already have a SEQ account? Link it" (`POST /auth/link` with an old invite code
  or session token).
- **The SDK** is vendored: `web/vendor/stytch-vanilla-js-6.3.0.headless.js`, the
  `dist/iife/index.headless.js` of the npm package `@stytch/vanilla-js` 6.3.0
  (MIT), unchanged. To update it, `npm pack @stytch/vanilla-js@<version>`, copy
  that file under its new version's name and change the three pages' `<script>`
  lines (`scripts/web-signin-check.mjs` holds them to the pinned name). No CDN
  is used, so there is no SRI to keep.
- **Stytch project settings** (Test and Live each): *Redirect URLs*: one per host,
  `<origin>/auth/callback`, for login and signup, and for both the OAuth and the Email
  magic link types. Stytch matches them exactly, so the host must be spelled as the
  browser shows it:
  - `http://localhost:8480/auth/callback` (`make play-web`);
  - `https://localhost:8443/auth/callback` (the dev Caddy setup);
  - `https://play.spielengine.com/auth/callback` (live).

  Add the same three origins as authorized domains (*Frontend SDK*): `http://localhost:8480`,
  `https://localhost:8443` and `https://play.spielengine.com`. Enable Google OAuth and the
  Email OTP and Email magic link products, and **leave PKCE off for email magic links**
  (it is a project setting: "PKCE required for Email Magic Links"), so that a link works
  without the browser having kept a secret. A link that still does not work (used,
  expired, or the project requires PKCE and it was opened elsewhere) ends on a page that
  says so and offers a code instead.
- **The callback page.** seqd serves `/auth/callback` (and `/activate`) from `-web-dir`,
  so every host above has it. The page still falls back to finishing the sign-in at `/`
  (which takes the same `?stytch_token_type=…&token=…`) when a host does not serve the
  callback file; that fallback is code only, and needs no Redirect URL.
- **Sessions.** The Stytch session lasts what the SDK gives; the seqd session (90
  days) is what return visits use. **Sign out** forgets it and ends the Stytch
  session. A saved session the server turns down at hello is dropped, with a
  message to sign in again.
- **`/activate`** takes `?code=ABCD-EFGH` (`verification_uri_complete`), asks the
  player to sign in if the browser has no Stytch session, then shows "Approve
  sign-in for seq-tui on this device? Code …, account …" with Approve and Deny.
  Behind seqd's `-web-dir` it is served from `activate/index.html`.
- **Checks.** `scripts/web-signin-check.mjs` (part of `make web-check`, so of
  `make verify`) runs the flow against a stubbed `/auth/*` and a stubbed SDK:
  Google, link, code, linking, saved sessions and their expiry, sign-out, the
  callback (a link that works, and one that fails and offers a code) and
  `/activate`.

## Local play in one command

```sh
make play-web          # http://localhost:8480/ (needs docker, as make play does)
```

seqd serves the page itself (`-web-dir web/dist -plain-http`, loopback), so no
Caddyfile, certificate or token hunt: the command builds the page, starts seqd on
the kept world, and prints the address and, with sign-in off, the development
token to paste. To try real sign-in on a Stytch **Test** project, export
`SEQ_STYTCH_PROJECT_ID`, `SEQ_STYTCH_DOMAIN` (`https://….customers.stytch.dev`),
`SEQ_STYTCH_PUBLIC_TOKEN` and `SEQ_STYTCH_SECRET` first, with the redirect URLs
above in the project (the next section has the commands). `make play-web` and `make play` share the database
container: run one at a time.

### Try real sign-in locally

Use a Stytch **Test** project with `http://localhost:8480/auth/callback` as a
Redirect URL and `http://localhost:8480` as an authorized domain (above). Open the
page as `http://localhost:8480/`, not `127.0.0.1`: the host must match.
`SEQ_STYTCH_SECRET` is a secret, so read it without echoing it.

bash or zsh:

```sh
export SEQ_STYTCH_PROJECT_ID=project-test-…
export SEQ_STYTCH_DOMAIN=https://….customers.stytch.dev
export SEQ_STYTCH_PUBLIC_TOKEN=public-token-test-…
read -rs "SEQ_STYTCH_SECRET?Stytch secret: " && export SEQ_STYTCH_SECRET   # zsh; in bash: read -rsp "Stytch secret: " SEQ_STYTCH_SECRET; export SEQ_STYTCH_SECRET
make play-web
```

fish:

```fish
set -x SEQ_STYTCH_PROJECT_ID project-test-…
set -x SEQ_STYTCH_DOMAIN https://….customers.stytch.dev
set -x SEQ_STYTCH_PUBLIC_TOKEN public-token-test-…
read -s -x SEQ_STYTCH_SECRET -P "Stytch secret: "
make play-web
```

`SEQ_STYTCH_SECRET` is optional for the page to load, but seqd needs it to check
the session the page posts, so set it to sign in end to end. With none of the
variables set, the page shows the development token form. To check the device
login against the same server, run
`seq-tui login -url ws://localhost:8480/seq/v1`: it prints an
`http://localhost:8480/activate?code=…` link to approve in the same browser.

## Serving it on the same origin as seqd

seqd checks the WebSocket's `Origin` against the request's `Host`, so the page
and `/seq/v1` must share a host. The page is static files; a proxy puts the two
under one name, with no server change. A Caddyfile that does it:

```caddyfile
play.example.org {
	encode                       # for what is not precompressed
	handle /seq/v1* {
		reverse_proxy https://127.0.0.1:8443 {
			header_up Host {hostport}   # Caddy rewrites Host for an https upstream; seqd compares Origin with Host
			transport http {
				tls_server_name seqd.internal   # or tls_insecure_skip_verify if seqd keeps its in-memory certificate
			}
		}
	}
	handle {
		root * /srv/seq-tui/web/dist
		file_server {
			precompressed br gzip   # serves seq-tui.wasm.br etc.
		}
		header /*.wasm Content-Type application/wasm
	}
}
```

For an `https://` upstream Caddy rewrites `Host` to the upstream's address, so
`header_up Host {hostport}` puts the public name back (`{hostport}`, not `{host}`:
seqd's `Origin` check compares the port too). Caddy handles WebSocket upgrades
without more settings. If seqd runs behind the proxy on plain HTTP the
`transport` block and the `header_up` go away; seqd serves TLS (`-listen`, `-cert`, `-key`), so it
is left in here. `application/wasm` matters: `WebAssembly.instantiateStreaming`
needs it (the page falls back to a slower load without).

## Local development

`make play-web` (above) is the simple way. This section is for Caddy with a
trusted certificate (it is what `https://localhost:8443` and the sign-in redirect URLs
there need). A browser needs a trusted certificate for `wss://`. Once:

```sh
brew install mkcert caddy nss
mkcert -install        # trusts mkcert's CA; without it Chrome shows ERR_CERT_AUTHORITY_INVALID
```

Restart Chrome after `mkcert -install`. Then, from the seq-tui repository root:

```sh
mkdir -p tmp
mkcert -cert-file tmp/localhost.pem -key-file tmp/localhost-key.pem localhost 127.0.0.1
make web
caddy run --config docs/Caddyfile.dev --adapter caddyfile   # the page and the proxy, on https://localhost:8443
```

(`--adapter caddyfile` is needed because the file is not named `Caddyfile`.) In
another terminal, from the **seq** repository, start seqd on a private port with
the same certificate (absolute paths) and the packs:

```sh
go run ./cmd/seqd -listen 127.0.0.1:9443 \
  -cert /abs/path/to/seq-tui/tmp/localhost.pem -key /abs/path/to/seq-tui/tmp/localhost-key.pem \
  -packs ../seq-world/packs -dev-account me      # prints a session token
```

The dev account's characters are `me-1` and `me-2`. Open `https://localhost:8443/`,
paste the token and leave Character id empty to pick one (or type `me-1`).
`make web` again after a change; a reload loads the new wasm.

`docs/Caddyfile.dev` serves `web/dist` on `https://localhost:8443`, proxies `/seq/v1` and
`/auth/` to seqd, and serves the `/auth/callback` page itself (see the file).

`127.0.0.1` rather than `localhost` keeps the upstream off `::1`, where seqd is not
listening; `tls_server_name localhost` is the name on the certificate.

Check the proxy and the Origin check before opening the browser; this prints `101`:

```sh
curl -sk -o /dev/null -w '%{http_code}\n' https://localhost:8443/seq/v1 \
  -H 'Connection: Upgrade' -H 'Upgrade: websocket' -H 'Sec-WebSocket-Version: 13' \
  -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' -H 'Origin: https://localhost:8443'
```

Troubleshooting:

- **ERR_CERT_AUTHORITY_INVALID**: the certificate is not trusted. Run `mkcert -install`, make the certificate again, restart Chrome.
- **403 "origin not authorized"** (the curl prints 403): `Host` does not match `Origin`. Check the Caddyfile has `header_up Host {hostport}` (with the port) and that you open the page as `https://localhost:8443`.
- **502 or a connection refused**: seqd is not running on `127.0.0.1:9443`, or the proxy points at `localhost` (which may resolve to `::1`).
- **"SEQ stopped: That character isn't on this account"** (`not_permitted` from the server): wrong character id. Leave Character id empty to pick one, or use `me-1` / `me-2`.

## How it is put together

- `third_party/bubbletea`: Bubble Tea v2.1.0 with a `tty_js.go` that gives it a js
  build (docs/upstream-bubbletea-js.md has the patch and the upstream pull request text).
- `app.Launch` (app/start.go) is the start-up code `cmd/seq-tui` and `cmd/seq-tui-web` share.
- `cmd/seq-tui-web/main_js.go` is the browser entry point: input from xterm's
  `onData` through an `io.Pipe` (the page's call never blocks), output to
  `term.write`, the size from the fit add-on, no signal handler, true colour.
- `web/`: `index.html`, `app.js`, the vendored xterm.js and fonts (`web/vendor`), `wasm_exec.js`
  from the Go in use (`$(go env GOROOT)/lib/wasm/`; copy it again with a new Go), and `keys.json`.
- `scripts/web-smoke.mjs` runs the wasm under Node on a recorded log: the first
  screen at 80 × 24, a resize, and `q` to quit. The log comes in through the page
  global `seqReplay`, which only the test build reads (`replay_js.go`, tag
  `webtest`, built by `make web-smoke`); the wasm that `make web` serves has no
  such hook, and `make web-nohook` fails if a wasm has the string.

## Why `convertEol` is set

There is no TTY in the browser, so Bubble Tea turns on its newline mapping
(`mapNl`, third_party/bubbletea/tea.go) and its renderer assumes that a line feed
also returns the cursor to column 0. xterm.js moves only down on a line feed
unless `convertEol` is on, so without it every later relative column move lands
too far right: the selected picker row is redrawn mid-screen, the picker is not
cleared, and lines overlap. `web/app.js` sets `convertEol: true`.

`make web-check` (part of `make verify`) guards it: it feeds the wasm's output
into a headless xterm.js (`scripts/vendor/xterm-headless.js`, 6.0.0) at 120 × 40
and compares the terminal's rows with the frame the program meant to draw (the
picker, after Down twice, Ctrl+L, the replayed game, after keys, after a resize to
100 × 30). It also runs with `convertEol` off, which must fail. The frame comes
from `seqHarness` and `seqLastView` (`cmd/seq-tui-web/harness_js.go`, tag
`webtest`); `make web-nohook` fails if the served wasm has them.

A resize that arrives before the program has started (while it loads) is applied
when it starts: the terminal's size is sent once as the hook is installed.

## Which terminals deliver Option-click (D109)

The game walks on Option-click (Alt-click) only. The program reads the Alt bit of
the SGR mouse report; the question is whether the terminal sends it.

| Terminal | Option-click reaches the game | How it was checked |
| --- | --- | --- |
| This page (xterm.js, 6.0) | yes, with `macOptionClickForcesSelection: false`; Alt on Windows and Linux | the option settings and the Alt bit are tested (`make web-check`); xterm.js's own encoder was not driven (no browser may be started) |
| macOS Terminal.app | not checked; Option-click is likely kept for cursor placement | not checked, no GUI application may be launched |
| iTerm2 | not checked; turn off "Option+click moves cursor" (Settings > General > Selection) | not checked |
| A Linux terminal | not checked; kitty, alacritty, wezterm, gnome-terminal and konsole report Alt with the mouse as far as their documentation goes | not checked |

Where Option-click does not arrive, click the thing (or the ground) and press **W**:
it walks to what was clicked, else the target, else the spot clicked.

## Playing on play.spielengine.com

In the browser: open https://play.spielengine.com/, sign in with Google or an
emailed link (or code), and play; the next visit shows Welcome back and one Play
button. In a terminal: `seq-tui login -url wss://play.spielengine.com/seq/v1`
prints a code and opens `/activate`; approve it there and the terminal saves the
sign-in (`~/.config/seq/play.spielengine.com.session`, mode 0600) and later runs
need no login. `seq-tui logout` forgets it. The README has the steps.

## Served by seqd behind a TLS proxy (M11)

`seqd -web-dir <web/dist> -plain-http` serves this page and `/seq/v1` from one address, with a proxy (HAProxy on play.spielengine.com) ending TLS in front of it. The page takes its server from `location`: `wss://` when the page came over https, `ws://` over http, and the host it came from, so no host name is built in. `?url=` still points a development page at another server.
