.PHONY: verify golden play

# verify is the repo's gate (seq-design D15): gofmt check, vet, test. Under 60 s.
verify:
	@start=$$(date +%s); \
	unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi; \
	go vet ./... || exit 1; \
	go test ./... || exit 1; \
	echo "make verify: OK in $$(( $$(date +%s) - start )) s"

# golden rewrites the golden screens in testdata/screens from the recorded logs.
# Review the diff before committing: a golden changing means the screen changed.
# Tests and `make verify` only compare, never write.
golden:
	go run ./cmd/mkscreens

# play starts a local seq server (PostgreSQL through docker, OrbStack's) and the
# TUI on it; see scripts/play.sh for what it needs. Quitting stops both.
play:
	scripts/play.sh
