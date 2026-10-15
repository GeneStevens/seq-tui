# seq-tui

`seq-tui` is the terminal client for SEQ. It is a thin client (seq-design `docs/07`): it renders state it is given and sends explicit player intents, and owns no gameplay truth.

Design lives in the `seq-design` repo. This repo is at milestone M6a: the client and the model, no screen yet (the screen is next).

- `model/`: a pure Go type built only from protocol messages (snapshot, events, intent results, zone map): you, entities in view, drops, encounters, the map, a bounded event log that keeps each event's reason, and the status of recent intents. It tracks the per-character stream number and takes each event exactly once (D39). `Text()` prints it as plain text.
- `client/`: the WebSocket transport, a `Session` that logs in with a human session token, selects, feeds the model, reconnects and resumes, and one command per intent (`Move`, `Stop`, `Target`, `Attack`, `Loot`, `UseDoor`, `Logout`) with `Result` matching each `intent_result`.
- `record/`: records a session's incoming messages as a log (one protobuf-JSON message per line) and reads it back.
- `testdata/logs/`: recorded logs the tests replay into the model with no backend. Tests only read them; `go run ./cmd/mkfixtures testdata/logs` rewrites the four built ones by hand. `live_login.jsonl` was recorded from a local `seqd`.
- `cmd/seq-tui`: connects, selects and prints the model.

The protocol types come from the `seq` module (`protocol/seqv1`), pinned in `go.mod`; `seq` is private, so fetch with `GOPRIVATE=github.com/GeneStevens/*` and git over ssh.

```
SEQ_SESSION_TOKEN=seq_st_... go run ./cmd/seq-tui -url wss://127.0.0.1:8443/seq/v1 -ca cert.pem -character me-1 [-record log.jsonl]
go run ./cmd/seq-tui -replay testdata/logs/fight_loot.jsonl
```

## Verify

```
make verify
```

Runs a `gofmt` check, `go vet ./...` and `go test ./...`, then prints the elapsed time. It must stay well under 60 seconds.

## History

The pre-reset code is preserved under the annotated tag `pre-reset` (commit `9d04d89`). `main` was restarted fresh from it; history is kept. See `PROVENANCE.md` for what is excluded.

The old top-down map rasterizer (`mapgeo.go` at `pre-reset`) is the planned starting point for the map view in M6. It is not ported yet.
