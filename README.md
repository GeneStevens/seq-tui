# seq-tui

`seq-tui` is the terminal client for SEQ. It is a thin client (seq-design `docs/07`): it renders state it is given and sends explicit player intents, and owns no gameplay truth.

Design lives in the `seq-design` repo. This repo is at milestone M6: a human can play the placeholder zone in a terminal against a local server.

- `model/`: a pure Go type built only from protocol messages (snapshot, events, intent results, zone map): you, entities in view, drops, encounters, the map, a bounded event log that keeps each event's reason, and the status of recent intents. It tracks the per-character stream number and takes each event exactly once (D39). `Text()` prints it as plain text.
- `client/`: a `Session` on the agent toolkit's connection (`agentkit` from the `seq` module, which owns the WebSocket, the login and the resume); its raw-message hook records each message and feeds the model. One command per intent (`Move`, `Stop`, `Target`, `Attack`, `Loot`, `UseDoor`, `Release`, `Logout`) with `Result` matching each `intent_result`. There is no transport of the TUI's own.
- `record/`: records a session's incoming messages as a log (one protobuf-JSON message per line) and reads it back.
- `testdata/logs/`: recorded logs the tests replay into the model with no backend. Tests only read them; `go run ./cmd/mkfixtures testdata/logs` rewrites the four built ones by hand. `live_login.jsonl`, `live_release.jsonl` (die, release, respawn) and `live_door.jsonl` (cross the zone line to the second zone and back) were recorded from a local `seqd`; `go run ./cmd/mkcapture` makes such a capture.
- `screen/`: the screen as a pure function from the model to exactly W x H cells (minimum 80x24): the top-down map (walls, bounds, doors, you, mobs with your target highlighted, drops), the You, Target and Nearby panels, the event log with the reason on every denial, the status line and the help panel. Colour only repeats what glyphs and words already say, and nothing shows what drives another character (D17).
- `app/`: the Bubble Tea v2 program: keys to the client's commands, the model to the screen.
- `testdata/screens/`: golden screens at 80x24 and 140x45, one set per recorded log. Tests only compare; `make golden` rewrites them (review the diff).
- `cmd/seq-tui`: the program. `cmd/mkscreens`: what `make golden` runs.

The protocol types and the connection come from the `seq` module (`protocol/seqv1`, `agentkit`), pinned in `go.mod`.

### Private module access (one time)

`seq` is private. Go must skip the public proxy for it and fetch it with git over ssh:

```
go env -w GOPRIVATE='github.com/GeneStevens/*'
git config --global url."git@github.com:".insteadOf "https://github.com/"
ssh -T git@github.com            # should greet you by name
```

`make play` downloads the module itself (`go mod download`) before it asks Go where it is. If the download fails it retries once with that same setup applied for the run only, and if that fails too it prints the go error and the three lines above and stops; it never carries on with an empty path. It also checks that docker is running.

```
make play                      # a local seq server on PostgreSQL (docker), and the TUI on it
SEQ_SESSION_TOKEN=seq_st_... go run ./cmd/seq-tui -insecure -url wss://127.0.0.1:8443/seq/v1 -character me-1 [-record log.jsonl]
go run ./cmd/seq-tui -replay testdata/logs/fight_loot.jsonl [-size 140x45 | -text]
```

`-insecure` accepts the development server's in-memory certificate and works for loopback addresses only. `make play` is `scripts/play.sh`: it starts the throwaway PostgreSQL from seq's `deploy/docker-compose.yml`, builds the pinned `seqd`, starts it with a development account, runs the TUI, and stops both when you quit. It needs docker (OrbStack's) and the private-module access above. The server has both placeholder zones, so from the test zone you can walk to the zone line in the east (`D` on the map; `Nearby` lists it) and press **E** there to cross, and **E** again at the other side to come back.

Keys (`?` shows them in the program): arrows or WASD move; Tab cycles targets (Esc clears); F or Space toggles auto-attack; L loots the nearest drop; E uses the nearest door; R releases a dead character (it respawns after a short wait; the You panel says so); O logs out; Q quits the screen and leaves the character in the world; V also shows movement in the event log.

## Verify

```
make verify
```

Runs a `gofmt` check, `go vet ./...` and `go test ./...`, then prints the elapsed time. It must stay well under 60 seconds.

## History

The pre-reset code is preserved under the annotated tag `pre-reset` (commit `9d04d89`). `main` was restarted fresh from it; history is kept. See `PROVENANCE.md` for what is excluded.

The map view uses the technique of the old rasterizer (`mapgeo.go` at `pre-reset`): line-to-cell rasterization in a player-centred, snapped viewport. It is written again here against the protocol's zone map; none of its code or EverQuest-derived data is carried (see `PROVENANCE.md`).
