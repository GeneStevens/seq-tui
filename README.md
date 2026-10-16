# seq-tui

`seq-tui` is the terminal client for SEQ. It is a thin client (seq-design `docs/07`): it renders state it is given and sends explicit player intents, and owns no gameplay truth.

Design lives in the `seq-design` repo. This repo is at milestone M7: a human can play the placeholder zone in a terminal against a local server, open an instance, walk in, and play it, including turn-based rounds with plans.

- `model/`: a pure Go type built only from protocol messages (snapshot, events, intent results, zone map): you, entities in view, drops, encounters, the map, a bounded event log that keeps each event's reason, and the status of recent intents. It tracks the per-character stream number and takes each event exactly once (D39). `Text()` prints it as plain text.
- `client/`: a `Session` on the agent toolkit's connection (`agentkit` from the `seq` module, which owns the WebSocket, the login and the resume); its raw-message hook records each message and feeds the model. One command per intent (`Move`, `Stop`, `Target`, `Attack`, `Loot`, `UseDoor`, `Release`, `Logout`) with `Result` matching each `intent_result`. There is no transport of the TUI's own.
- `record/`: records a session's incoming messages as a log (one protobuf-JSON message per line) and reads it back.
- `testdata/logs/`: recorded logs the tests replay into the model with no backend. Tests only read them. `fight_loot`, `kill_steal`, `crossing` and `drop_resume` are built by hand (`go run ./cmd/mkfixtures testdata/logs`); `play`, `release`, `door`, `solo_trial`, `quick_turns` and `duo_turns` are the sessions of the client scenarios, recorded by them (`make fixtures`). A scenario writes `# mark NAME` lines into its log at the points it names; the render tests and `model` tests replay a log up to a mark (`file@mark`), so a regenerated log moves nothing.
- `scenario/`: the client scenarios (below).
- `screen/`: the screen as a pure function from the model to exactly W x H cells (minimum 80x24): the top-down map (walls, bounds, doors, you, mobs with your target highlighted, drops), the You, Target and Nearby panels, the event log with the reason on every denial, the status line and the help panel. Colour only repeats what glyphs and words already say, and nothing shows what drives another character (D17).
- `app/`: the Bubble Tea v2 program: keys to the client's commands, the model to the screen.
- `testdata/screens/`: golden screens at 80x24 and 140x45 of the recorded logs, replayed to a mark. `testdata/scenarios/`: golden screens drawn by the real program in a scenario, at the same two sizes. Tests only compare; `make golden` rewrites both (review the diff).
- `cmd/seq-tui`: the program. `cmd/mkscreens`: half of what `make golden` runs. `cmd/mkfixtures`: the hand-built logs.

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

## Instances and rounds

**I** opens the zone's instance menu, read from the zone map's presets: each shows its name, its time mode (real-time, or turn-based with the round length in ticks and the time the players have to plan) and its party cap. Up and Down or the number choose, Enter sends `instance_open`, Esc closes. The answer is in the event log in words (`instance_opened`, or `instance_open_failed` with why), and every denial reason is given as words with its code in brackets. Walk to the preset's door (`gate-north` in the test zone) and press **E**, as at any door; **E** at the door again leaves. The map and panels follow the handoff. An **Instance** panel shows what the server says the instance is (D49: the snapshot's `instance`, never the space's name): the preset, its time mode, party cap, difficulty and, for a turn-based one, its sleep limit; or its state when it is open, or ended with why (cleared, abandoned or expired, followed by the hand-back to the zone). The menu shows each preset's sleep limit as well.

In a **turn-based** instance a bare intent is refused by the server (`plan_required`), so the keys build a **plan** instead and send nothing until you submit it. The **Round** panel's title is the round and a live countdown to its deadline; the panel says "N of M submitted" (the server's `participants`, which can rise while a party gathers in round 1; the server does not announce another character's plan, so the count is what its last round event said plus your own receipt) and lists the plan being built, and the map marks `+` where the plan's moves end.

| key | adds to the plan |
| --- | --- |
| arrows, W A S D | a move step, 3 m on from where the plan's last move ends |
| Tab | the next target (from the plan's target) |
| F or Space | auto-attack on or off (from the plan's state) |
| L | loot the drop nearest where the plan ends |
| E | use the nearest door |
| X, R | stop; release |

**Enter** submits the plan (at most 16 steps; an empty plan means "do nothing"), **Backspace** removes the last step, **Esc** clears it. The receipt says "plan accepted, counts even if you disconnect"; Enter again replaces the plan until the round runs. A plan sent after the deadline is denied with `round_closed`: plan for the next round. When a round runs, the log says why (`all_submitted`, `deadline` or `instance_ended`), "N of M submitted", and exactly what each character played: its own plan, a stand-in plan or the safe default (D49's `plays`). A new round starts an empty plan. After a reconnect (the instance may have slept) the fresh snapshot and the repeated `round_started` restore the Round panel, and a plan already accepted shows as submitted.

On a small screen (80x24) the right column keeps what fits: inside an instance the Nearby panel (and in a turn-based one the Target panel) is left out to make room for the Instance and Round panels.

Keys (`?` shows them in the program): arrows or WASD move; Tab cycles targets (Esc clears); F or Space toggles auto-attack; L loots the nearest drop; E uses the nearest door; R releases a dead character (it respawns after a short wait; the You panel says so); I opens the instance menu; O logs out; Q quits the screen and leaves the character in the world; V also shows movement in the event log.

## Client scenarios

No test of the TUI needs a person (D49). `scenario/` drives the real program: a real `tea.Program` runs `app.App` with its input a pipe the scenario writes key bytes to, on a real `client.Session`, against `seq`'s in-process test server (`testserver.Start`). The scenario moves the server's clock with `Advance`; nothing sleeps for game time. A scenario is a short script:

```go
w := scenario.New(t, testserver.Options{...})
w.Place("ann", "aria", 100000, 89000)
p := w.Login("ann", "aria")
p.Press("i", "4", "enter")       // keys, one at a time; each waits for the program and the server
w.Advance(time.Second)           // the clock, not a sleep
p.Press("e")
p.Eventually("the instance", inside) // what the server does outside the clock (a handoff)
p.Model(func(m *model.Model) { ... })
p.Golden("round_plan")           // the screen at both sizes against testdata/scenarios, and a mark in the recording
```

The scenarios (`scenario/flows_test.go`): login, move, target, attack, kill and loot; die, release and respawn; the door across and back; solo-trial from the menu to the hand-back; quick-turns round by round (plans edited with Backspace and Esc, the receipt, a missed deadline, a dropped and resumed connection, a fresh login into the open round, clear); a duo on one account entering together in the gathering round.

Speed gates (D49), enforced in the test code: each scenario under 2 s, the whole suite under 15 s (`scenario.Main` in `TestMain`). `SEQ_SCENARIO_SLOW=3s go test ./scenario` makes each scenario wait 3 s at its end and shows the gate failing. The suite's gate counts the whole run, so it means something with `-count=1`.

`make fixtures` runs the scenarios with `SEQ_FIXTURES` set, which makes the scenarios that declare a fixture write their recording to `testdata/logs`, and rebuilds the hand-built logs. Run `make golden` after it, and review both diffs.

## Verify

```
make verify
```

Runs a `gofmt` check, `go vet ./...` and `go test ./...` (which includes the scenarios), then prints the elapsed time. It must stay well under 60 seconds.

## History

The pre-reset code is preserved under the annotated tag `pre-reset` (commit `9d04d89`). `main` was restarted fresh from it; history is kept. See `PROVENANCE.md` for what is excluded.

The map view uses the technique of the old rasterizer (`mapgeo.go` at `pre-reset`): line-to-cell rasterization in a player-centred, snapped viewport. It is written again here against the protocol's zone map; none of its code or EverQuest-derived data is carried (see `PROVENANCE.md`).
