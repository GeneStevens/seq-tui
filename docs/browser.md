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

- Asks for a session token (the one `seq-tui` takes in `SEQ_SESSION_TOKEN`), and
  optionally a character id, or to make a new character. The token is kept in
  the tab's `sessionStorage`, so it survives a reload and goes when the tab
  closes. A real web login is a later project.
- Connects to `wss://<the page's host>/seq/v1`. Open the page as
  `…/?url=wss://other/seq/v1` to name another server (development: the server
  checks the browser's `Origin`, so another host is refused unless it is the page's).
- Keeps Ctrl+A, Ctrl+K, Ctrl+L, Ctrl+R, Ctrl+U and F1 from the browser
  (`preventDefault`), so they reach the game. The list is `web/keys.json`;
  `go test ./web` holds it to the help's binding table, so a new Ctrl+letter or
  function-key binding that is not listed fails. Ctrl+W, Ctrl+T and Ctrl+N cannot
  be taken from a browser: no binding may use them (the same test).
- Asks before the tab is closed or reloaded while the game runs (`beforeunload`).
- On macOS, Option is Alt (`macOptionIsMeta`), and a click does not move xterm's cursor.
- Says so under 80 × 24 (the program shows its own too small note as well).
- Draws box and block characters with xterm's `customGlyphs` (the WebGL renderer;
  without WebGL the DOM renderer is used and they come from the font), in a bundled
  DejaVu Sans Mono, so that `⚠`, `✉` and the map glyphs are one cell wide.

Not in the browser build: duo mode (`-duo`), session logs (it runs as
`-no-record`: the sessions view lists the server's own log only) and
`-replay`. Key releases do not exist in xterm.js, so the walk uses the same
repeat-idle fallback as a terminal without the kitty keyboard protocol
(xterm.js never answers the query, so the program does not expect releases).

## Serving it on the same origin as seqd

seqd checks the WebSocket's `Origin` against the request's `Host`, so the page
and `/seq/v1` must share a host. The page is static files; a proxy puts the two
under one name, with no server change. A Caddyfile that does it:

```caddyfile
play.example.org {
	encode                       # for what is not precompressed
	handle /seq/v1* {
		reverse_proxy 127.0.0.1:8443 {
			transport http {
				tls
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

Caddy keeps the `Host` header when it proxies and handles WebSocket upgrades
without more settings. If seqd runs behind the proxy on plain HTTP the
`transport` block goes away; seqd serves TLS (`-listen`, `-cert`, `-key`), so it
is left in here. `application/wasm` matters: `WebAssembly.instantiateStreaming`
needs it (the page falls back to a slower load without).

## Local development

A browser needs a trusted certificate for `wss://`. With [mkcert](https://github.com/FiloSottile/mkcert):

```sh
mkcert -install
mkcert -cert-file tmp/localhost.pem -key-file tmp/localhost-key.pem localhost 127.0.0.1

# 1. seqd on a private port, with that certificate
seqd -listen 127.0.0.1:9443 -cert tmp/localhost.pem -key tmp/localhost-key.pem -dev-account me   # prints a session token

# 2. the page and the proxy, on https://localhost:8443
make web
caddy run --config docs/Caddyfile.dev
```

`docs/Caddyfile.dev` serves `web/dist` on `https://localhost:8443` and proxies `/seq/v1` to seqd:

```caddyfile
{
	auto_https disable_redirects
}
localhost:8443 {
	tls tmp/localhost.pem tmp/localhost-key.pem
	handle /seq/v1* {
		reverse_proxy https://localhost:9443
	}
	handle {
		root * web/dist
		file_server
		header /*.wasm Content-Type application/wasm
	}
}
```

Open `https://localhost:8443/` and paste the token. `make web` again after a change;
a reload loads the new wasm.

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
  screen at 80 × 24, a resize, and `q` to quit.
