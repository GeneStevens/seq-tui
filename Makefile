.PHONY: verify golden fixtures play

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

# play starts a local seq server (PostgreSQL through docker, OrbStack's) and the
# TUI on it; see scripts/play.sh for what it needs. Quitting stops both.
play:
	scripts/play.sh
