.PHONY: verify golden fixtures play play-testgrounds play-reset web web-smoke

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
	SEQ_GOLDEN=1 go test ./screen -run TestChatGoldens -count=1

# play starts a local seq server (PostgreSQL through docker, OrbStack's) that
# serves Thornwick and keeps its data between runs, and the TUI on it, which asks
# which character to play; see scripts/play.sh for what it needs and the README
# for where the data lives. Quitting stops both and keeps the world.
play:
	scripts/play.sh

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
	cp web/index.html web/app.js web/keys.json web/wasm_exec.js $(WEBDIST)/
	cp -R web/vendor $(WEBDIST)/vendor
	node scripts/web-compress.mjs $(WEBDIST)

# web-smoke builds the wasm (a fresh one, outside web/dist) and runs it in node
# with piped input and output: the first screen at 80x24, a resize, a key to quit.
# It also holds the browser build to compiling, which go build ./... does not (it
# builds for this system). Skipped, with a note, where node is not installed.
web-smoke:
	@mkdir -p $(CURDIR)/tmp/websmoke
	GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o tmp/websmoke/seq-tui.wasm ./cmd/seq-tui-web
	@if command -v node >/dev/null; then \
		cp web/wasm_exec.js tmp/websmoke/ && node scripts/web-smoke.mjs tmp/websmoke; \
	else echo "web-smoke: node not found, the wasm builds but was not run"; fi
