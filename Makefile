.PHONY: verify

# verify is the repo's gate (seq-design D15): gofmt check, vet, test. Under 60 s.
verify:
	@start=$$(date +%s); \
	unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi; \
	go vet ./... || exit 1; \
	go test ./... || exit 1; \
	echo "make verify: OK in $$(( $$(date +%s) - start )) s"
