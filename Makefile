.PHONY: verify golden fixtures play play-web play-testgrounds play-reset web web-smoke web-check web-wasm-test web-nohook release-mac

# verify is the repo's gate (seq-design D15): gofmt check, vet, test. Under 60 s.
# The scenarios read seq-world's packs from the module cache, which no import
# fills (only a tool and the test server's packs path name it), so it is
# downloaded first; it is a private module (see the README).
verify:
	@start=$$(date +%s); \
	unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi; \
	go mod download github.com/GeneStevens/seq-world || exit 1; \
	go vet ./... || exit 1; \
	go test ./... || exit 1; \
	$(MAKE) --no-print-directory web-smoke || exit 1; \
	$(MAKE) --no-print-directory web-check || exit 1; \
	echo "make verify: OK in $$(( $$(date +%s) - start )) s"

# fixtures regenerates the recorded logs under testdata/logs: the hand-built ones
# (cmd/mkfixtures) and the sessions of the client scenarios, which write their
# own recording when SEQ_FIXTURES names the directory. Run `make golden` after
# it, and review both diffs.
fixtures:
	go run ./cmd/mkfixtures testdata/logs
	SEQ_FIXTURES=$(CURDIR)/testdata/logs SEQ_GOLDEN=1 go test ./scenario -count=1

# golden rewrites the golden screens: testdata/screens from the recorded logs,
# testdata/scenarios from the scenarios' own screens.
# Review the diff before committing: a golden changing means the screen changed.
# Tests and `make verify` only compare, never write.
golden:
	go run ./cmd/mkscreens
	SEQ_GOLDEN=1 go test ./scenario -count=1
	SEQ_GOLDEN=1 go test ./screen -run 'TestChatGoldens|TestTheBiggerDownsGoldens|TestQuestV2Goldens' -count=1

# play starts a local seq server (PostgreSQL through docker, OrbStack's) that
# serves Thornwick and keeps its data between runs, and the TUI on it, which asks
# which character to play; see scripts/play.sh for what it needs and the README
# for where the data lives. Quitting stops both and keeps the world.
play:
	scripts/play.sh

# play-web is make play for the browser: the same kept world, with seqd serving the
# browser build itself on http://localhost:8480 (no proxy, certificate or Caddyfile).
# It builds the page first (make web). Stytch settings in the environment pass
# through, to try the real sign-in; without them it prints a development token.
play-web: web
	SEQ_PLAY_WEB=1 scripts/play.sh

# play-testgrounds is the setup for tests and quick checks: Testgrounds (the
# engine's test zone), me-1, and a database thrown away on quitting.
play-testgrounds:
	SEQ_TESTGROUNDS=1 scripts/play.sh

# play-reset wipes the kept world (characters, items, progress) after asking.
play-reset:
	scripts/play-reset.sh

# web builds the browser version (seq-design D106; docs/browser.md): the program
# as WebAssembly (GOOS=js, on the bubbletea copy in third_party) and the page that
# draws it with xterm.js, in web/dist, each file also as .gz and .br. Its sizes
# are printed. Needs node (for the compression and the smoke test).
WEBDIST := web/dist
web:
	rm -rf $(WEBDIST) && mkdir -p $(WEBDIST)
	GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o $(WEBDIST)/seq-tui.wasm ./cmd/seq-tui-web
	cp web/index.html web/app.js web/auth.js web/signin-ui.js web/callback.js web/activate.js web/page.css web/keys.json web/wasm_exec.js $(WEBDIST)/
	cp -R web/activate web/auth $(WEBDIST)/
	cp -R web/vendor $(WEBDIST)/vendor
	@$(MAKE) --no-print-directory web-nohook WASM=$(WEBDIST)/seq-tui.wasm
	node scripts/web-compress.mjs $(WEBDIST)

# web-nohook fails if the wasm named by WASM (default: a fresh production build,
# in tmp/websmoke) has the seqReplay test hook, which only the webtest build has.
WASM ?=
web-nohook:
	@w="$(WASM)"; \
	if [ -z "$$w" ]; then mkdir -p $(CURDIR)/tmp/websmoke; w=tmp/websmoke/seq-tui-prod.wasm; \
		GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o $$w ./cmd/seq-tui-web || exit 1; fi; \
	for h in seqReplay seqHarness seqLastView; do \
		if grep -aq $$h "$$w"; then echo "web-nohook: $$w has the $$h test hook; it belongs behind -tags webtest"; exit 1; fi; done; \
	echo "web-nohook: $$w has no test hooks"

# web-wasm-test builds the wasm with the webtest tag, outside web/dist.
web-wasm-test:
	@mkdir -p $(CURDIR)/tmp/websmoke
	GOOS=js GOARCH=wasm go build -tags webtest -trimpath -ldflags='-s -w' -o tmp/websmoke/seq-tui.wasm ./cmd/seq-tui-web

# web-check runs the sign-in pages' check (scripts/web-signin-check.mjs: the page
# logic against stubs), then feeds the wasm's output into a headless xterm.js (120x40, as the page
# sets it up, with convertEol) and compares its rows with the frame the program
# meant to draw: the picker, a screen change, the game, keys, a resize. It also
# runs with convertEol off, which must fail: that is the drift web/app.js prevents.
web-check: web-wasm-test
	@if command -v node >/dev/null; then \
		node scripts/web-signin-check.mjs && \
		cp web/wasm_exec.js tmp/websmoke/ && \
		node scripts/web-check.mjs tmp/websmoke && \
		node scripts/web-check.mjs tmp/websmoke --convert-eol=off --expect-fail >/dev/null && \
		echo "web-check: rows match with convertEol, and differ without it"; \
	else echo "web-check: node not found, not run"; fi

# web-smoke builds the wasm (a fresh one, outside web/dist, with the webtest tag
# for the recorded-log hook that the served build lacks) and runs it in node
# with piped input and output: the first screen at 80x24, a resize, a key to quit.
# It also holds the browser build to compiling, which go build ./... does not (it
# builds for this system). Skipped, with a note, where node is not installed.
web-smoke: web-wasm-test
	@$(MAKE) --no-print-directory web-nohook
	@if command -v node >/dev/null; then \
		cp web/wasm_exec.js tmp/websmoke/ && node scripts/web-smoke.mjs tmp/websmoke; \
	else echo "web-smoke: node not found, the wasm builds but was not run"; fi

# release-mac builds seq-tui for darwin-arm64 and darwin-amd64 and packs a .tar.gz
# for each, with a README line and SHA256SUMS, in dist/release. It publishes
# nothing; the README has the `gh release create` command.
release-mac:
	scripts/release-mac.sh $(VERSION)
