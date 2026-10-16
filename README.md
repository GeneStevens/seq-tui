# seq-tui

`seq-tui` is the terminal client for SEQ. It is a thin client (seq-design `docs/07`): it renders state it is given and sends explicit player intents, and owns no gameplay truth.

Design lives in the `seq-design` repo. This repo is at milestone M8: a human can make a character of one of the six classes, play the placeholder zone in a terminal against a local server with the ability bar on keys 1 to 8, mana, casts, cooldowns, effects, levels and XP, open an instance, walk in, and play it, including turn-based rounds with plans; and (D50) the class menu from the server, who in the party has submitted, effects seen on login, slots changed between rounds, casts that land in a later round, gear, loot, crowns, vendors, trainers, the binder, the bank, trade between characters, and the Peril fee; and (M8 content) Thornwick and the Thornwick Downs: talking to the people, the contract board and the Contracts panel, timed world events and roaming rares, and a map that scrolls with the character and shows the whole zone on a key.

- `model/`: a pure Go type built only from protocol messages (snapshot, events, intent results, zone map): you, entities in view, drops, encounters, the map, a bounded event log that keeps each event's reason, and the status of recent intents. It tracks the per-character stream number and takes each event exactly once (D39). `Text()` prints it as plain text.
- `client/`: a `Session` on the agent toolkit's connection (`agentkit` from the `seq` module, which owns the WebSocket, the login and the resume); its raw-message hook records each message and feeds the model. One command per intent (`Move`, `Stop`, `Target`, `Attack`, `Loot`, `UseDoor`, `Release`, `UseAbility`, `SetSlot`, `Logout`), and `Connect`, `CreateCharacter` and `Select` for a new character with `Result` matching each `intent_result`. There is no transport of the TUI's own.
- `record/`: records a session's incoming messages as a log (one protobuf-JSON message per line) and reads it back.
- `testdata/logs/`: recorded logs the tests replay into the model with no backend. Tests only read them. `fight_loot`, `kill_steal`, `crossing` and `drop_resume` are built by hand (`go run ./cmd/mkfixtures testdata/logs`); `play`, `release`, `door`, `solo_trial`, `quick_turns`, `duo_turns`, `wizard`, `casts`, `warrior`, `levelup` and `turn_cast` are the sessions of the client scenarios, recorded by them (`make fixtures`). A scenario writes `# mark NAME` lines into its log at the points it names; the render tests and `model` tests replay a log up to a mark (`file@mark`), so a regenerated log moves nothing.
- `scenario/`: the client scenarios (below).
- `screen/`: the screen as a pure function from the model to exactly W x H cells (minimum 80x24): the top-down map (walls, bounds, doors, you, mobs with your target highlighted, drops), the You, Target and Nearby panels, the ability bar, the slot screen, the character creation screen, the event log with the reason on every denial, the status line and the help panel. Colour only repeats what glyphs and words already say, and nothing shows what drives another character (D17).
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

`-new` opens on the character creation screen instead of a character (`SEQ_NEW=1 make play`). `-insecure` accepts the development server's in-memory certificate and works for loopback addresses only. `make play` is `scripts/play.sh`: it starts the throwaway PostgreSQL from seq's `deploy/docker-compose.yml`, builds the pinned `seqd`, starts it with a development account, runs the TUI, and stops both when you quit. It needs docker (OrbStack's) and the private-module access above. The server has both placeholder zones, so from the test zone you can walk to the zone line in the east (`D` on the map; `Nearby` lists it) and press **E** there to cross, and **E** again at the other side to come back.

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
| 1 to 8 | use the ability in that slot (it needs no target of its own: the server uses your current target) |

A plan holds **one move or door and one ability or auto-attack toggle** (doc 05 §4.6), with the target steps they need: a second move, ability or toggle is not added, and the status line says why (Backspace removes the one the plan has). A cast in a plan plays through the round as in real time, and a move in the same plan cancels it. Slots are set out of play (**B**), not in a plan.

**Enter** submits the plan (at most 16 steps; an empty plan means "do nothing"), **Backspace** removes the last step, **Esc** clears it. The receipt says "plan accepted, counts even if you disconnect"; Enter again replaces the plan until the round runs. A plan sent after the deadline is denied with `round_closed`: plan for the next round. When a round runs, the log says why (`all_submitted`, `deadline` or `instance_ended`), "N of M submitted", and exactly what each character played: its own plan, a stand-in plan or the safe default (D49's `plays`). A new round starts an empty plan. After a reconnect (the instance may have slept) the fresh snapshot and the repeated `round_started` restore the Round panel, and a plan already accepted shows as submitted.

On a small screen (80x24) the right column keeps what fits: inside an instance the Nearby panel (and in a turn-based one the Target panel) is left out to make room for the Instance and Round panels, and with the ability bar the Round panel (the plan) comes before the Instance panel.

Keys (`?` shows them in the program): arrows or WASD move; Tab cycles targets (Esc clears); F or Space toggles auto-attack; 1 to 8 use the abilities in those slots; B sets the slots; L loots the nearest drop; E uses the nearest door; R releases a dead character (it respawns after a short wait; the You panel says so); I opens the instance menu; O logs out; Q quits the screen and leaves the character in the world; V also shows movement, rules ticks and mana in the event log; G talks to the nearest person; K opens the Contracts screen; M shows the whole zone.

## People, contracts, events and the map

**G** talks to the nearest person you have seen (the server decides whether they are within 5 m). Their lines and what they offer are on the **Talking** screen, with the options as the server sent them; an option opens the screen it leads to: buying and selling, training, binding and banking open that trader's window (the existing N screens), and taking or handing in a contract opens the **contract board**. A person with nothing to offer yet (the Watch Captain) says so. Up / Down or a number choose, Enter does it, Esc closes.

The **contract board** lists what it offers you now (name, kind, levels; Tab for the contracts you hold) with the description and the pay. Enter takes an offered contract, or hands in a held one; X gives a held one up. The client sends the intent and the server refuses with its reason (`contract_limit`, `contract_held`, `contract_not_done`, `out_of_range`). **K** opens the **Contracts** screen (and a Contracts panel appears in the right column when there is room): each contract held with its progress in words (kills, items carried, or "go to the marker" and then "marker reached"), where a scout's marker is, and the pay; X gives the selected one up. The server describes a held contract from the zone the character is in, and the Downs have no board, so the client keeps the name, text and pay it saw on the board and fills a held contract in from them.

A **world event** is announced to the zone: a banner row across the top says what is on and, for thirty seconds after it ends, how it ended; the log has every announcement, leak and payout. A **roaming rare** in view is marked `r` on the map, `RARE` in Nearby and the Target panel, and a log line says it has appeared. The protocol does not mark a mob as a rare, so the client knows them by name (`model.IsRare`).

The map scrolls: it is always centred on the character at a fixed scale, and its title says where in the zone that is. `&` marks a contract board, `?` someone who only talks, `$` the other traders, `%` the marker of a scout contract you hold, `D` a door, and `c` a camp (a place a mob was first seen; the client remembers it). The people are told only in a snapshot, so the client keeps the ones it has seen, in each space, and draws them wherever they stand. **M** shows the whole zone in the panel, with what has been seen of it.

## Classes and combat

`seq-tui -new` (or `SEQ_NEW=1 make play`) opens the **creation screen**: type a name (3 to 16 letters; the server makes its id the name in lower case and refuses a taken one with its reason), Tab or Down moves to the **class menu the server sent with its welcome** (the client has no class text of its own: each class's name, roles, description and the kit it starts with, free, all come from it; Up / Down or the number choose), Enter makes the character and plays it. A refusal (`bad_name`, `name_taken`, `unknown_class`, `too_many_characters`) is shown in words on the screen, which stays.

A character with abilities has an **ability bar** across the screen: eight slots, each with the ability, its mana cost on a wide screen, and one word for whether it can be used now: `ready`, `busy` (a cast is running), a cooldown's seconds left, `gcd` (the global cooldown), `mana`, `tgt?` (an enemy ability and no target) or `range`. The words are a hint from the model's own numbers; the key always sends the intent and the server decides, and its refusal (`no_mana`, `on_cooldown`, `out_of_range`, ...) is on the status line in words. The bar's last row lists the effects on you and on your target with the time left (a buff or damage over time counts rules ticks, so its time is marked `~`; a stun, root or snare says Stunned, Rooted, Snared). The You panel has the class, an HP bar, a mana bar for casters, an XP bar with what is left to the next level, and the **cast bar**: the ability and the time left. A cast that does not land (you moved, damage broke it, you were stunned, the target is gone) is a red `!` line in the event log and in the You panel until the next cast begins. A level-up is a `+` line in the log that names the new ability and its key. Mana changes are in the log under **V** only; the bar shows them.

**B** opens the **slot screen** (out of combat; the server refuses it in combat with `in_combat`): Up / Down or 1 to 8 choose a slot, Enter picks an ability from the list of what the character knows, Backspace empties the slot, Esc closes it. In a **turn-based** instance a slot changes with the plan (D50): the same keys add a step, the slot screen marks it `(in the plan)`, Enter on the main screen submits it, and the slot holds when the round opens (it is allowed in a fight there). A cast that will land in a later round says so (`lands in round N`) in the You panel, the Round panel and the log.

## Items, traders and trade

The screens below read only what the snapshot and the events give (the snapshot carries the item catalogue, the traders in view and the trade window); the server decides every action and its refusal is on the status line in words.

- **C** is the inventory and gear screen: the ten gear slots and the items carried, each with its tier, whether it is bound (`[bound]`; Rare and Named are bound on pickup, a traded item is marked `!` and bound to its owner) and what it adds. Tab switches lists, Enter wears the item or takes the gear off (out of combat).
- The **You** panel shows crowns at the right of its first line. Drops that the server marks as not yours to take (a Rare drop you are not entitled to) are `~` on the map and `not lootable` in Nearby, and **L** says so without asking the server.
- **N** lists the traders in view (`$` on the map): Enter opens a **vendor** (the stock on the left, what you carry on the right; Enter buys one or sells one), a **trainer** (the abilities of your class with level and cost; Enter learns the one under the cursor into the first empty slot), a **binder** (Enter sets the bind point) or the **bank** (Tab: deposit or withdraw).
- **T** is the trade window. With no window it lists the characters in view and in your group; Enter asks one, and the other presses T and Enter to agree. Then each side puts items in (Enter on what you carry), sets crowns (**C**, digits, Enter), takes things back (Tab to the offer, Enter), and **Y** confirms; any change clears both confirmations; **X** cancels. Bound items are refused by the server with `bound`.
- The instance menu scrolls, and marks the heroic (Peril) presets: the server takes an entry fee in crowns when they open (the protocol does not say how much; the log does, once it is paid, and a refund shows there too).

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

The class scenarios (`scenario/combat_test.go`) create their characters with keys on the creation screen (`w.NewCharacter`, `p.Type`) or log in a class character the test server makes (`Account.Classed`), and use `Options.Zone` to adjust the test server's copy of the zone data where a flow needs it (for instance, the level a class learns Root at, so a level 1 Wizard can root a mob; a mob's hit points or XP). They are: create a Wizard, slot abilities with the slot screen, root a scuttler and nuke it, see the mana drop and come back; a cast cancelled by moving and one interrupted by damage; a Warrior taunts, uses Second Wind, bashes and kills; a level-up (a level no longer grants abilities: the trainer sells them); a turn-based round with a cast in the plan; the creation screen's refusals; creating a character from the server's class menu; changing a slot between rounds; logging in mid-fight to see a rooted mob.

The economy scenarios (`scenario/economy_test.go`): kill the trophy scuttler, loot it and wear the loot while a bystander sees the Rare drop as not lootable; buy, sell, train, bind and bank at the traders; a trade between two characters with a bound item refused; open a Peril instance and pay the fee.

The Thornwick scenarios (`scenario/town_test.go`) run on `scenario.StartTown`, a test server built like seq's `testserver` (the same packages and manual clock) that runs Thornwick and the Thornwick Downs, which `testserver` does not: create a character, start in the square, talk to the board and take a Hunt (and the limit of three, and giving one up); walk out of the east gate, kill the wolves, come back and hand it in; buy from a vendor and train an ability; a Scout contract to a marker; the Harvest Raid announced and joined; Old Grimtusk coming into view.

The older scenarios (`scenario/flows_test.go`): login, move, target, attack, kill and loot; die, release and respawn; the door across and back; solo-trial from the menu to the hand-back; quick-turns round by round (plans edited with Backspace and Esc, the receipt, a missed deadline, a dropped and resumed connection, a fresh login into the open round, clear); a duo on one account entering together in the gathering round.

Speed gates (D49), enforced in the test code: each scenario under 2 s, the whole suite under 15 s (`scenario.Main` in `TestMain`). `SEQ_SCENARIO_SLOW=3s go test ./scenario` makes each scenario wait 3 s at its end and shows the gate failing. The suite's gate counts the whole run, so it means something with `-count=1`.

`make fixtures` runs the scenarios with `SEQ_FIXTURES` set, which makes the scenarios that declare a fixture write their recording to `testdata/logs`, and rebuilds the hand-built logs. Run `make golden` after it, and review both diffs. (`make fixtures` also rewrites the scenario goldens as it runs, so a screen that changed does not stop the recording half way.)

## Verify

```
make verify
```

Runs a `gofmt` check, `go vet ./...` and `go test ./...` (which includes the scenarios), then prints the elapsed time. It must stay well under 60 seconds.

## History

The pre-reset code is preserved under the annotated tag `pre-reset` (commit `9d04d89`). `main` was restarted fresh from it; history is kept. See `PROVENANCE.md` for what is excluded.

The map view uses the technique of the old rasterizer (`mapgeo.go` at `pre-reset`): line-to-cell rasterization in a player-centred, snapped viewport. It is written again here against the protocol's zone map; none of its code or EverQuest-derived data is carried (see `PROVENANCE.md`).
