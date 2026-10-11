# seq-tui

`seq-tui` is the terminal client for SEQ. It is a thin client (seq-design `docs/07`): it renders state it is given and sends explicit player intents, and owns no gameplay truth.

Design lives in the `seq-design` repo. This repo is at milestone M0: a Go skeleton with a banner `main.go`, a trivial test and `make verify`. No client code yet (that is M6).

## Verify

```
make verify
```

Runs a `gofmt` check, `go vet ./...` and `go test ./...`, then prints the elapsed time. It must stay well under 60 seconds.

## History

The pre-reset code is preserved under the annotated tag `pre-reset` (commit `9d04d89`). `main` was restarted fresh from it; history is kept. See `PROVENANCE.md` for what is excluded.

The old top-down map rasterizer (`mapgeo.go` at `pre-reset`) is the planned starting point for the map view in M6. It is not ported yet.
