# seq-tui

`seq-tui` is the terminal client for SEQ. It is a thin client (seq-design `docs/07`): it renders state it is given and sends explicit player intents, and owns no gameplay truth.

Design lives in the `seq-design` repo. This repo is at milestone M9 (M8 and the playing-together layer below): a human can make a character of one of the six classes, play the placeholder zone in a terminal against a local server with the ability bar on keys 1 to 8, mana, casts, cooldowns, effects, levels and XP, open an instance, walk in, and play it, including turn-based rounds with plans; and (D50) the class menu from the server, who in the party has submitted, effects seen on login, slots changed between rounds, casts that land in a later round, gear, loot, crowns, vendors, trainers, the binder, the bank, trade between characters, and the Peril fee; and (M8 content) Thornwick and the Thornwick Downs: talking to the people, the contract board and the Contracts panel, timed world events and roaming rares, and a map that scrolls with the character and shows the whole zone on a key; and (M8n) the Watch Captain's companion quest, choosing a recruit's role, a Companion panel with stances and orders, and a second character of the account played by a stance; and (D53, D54) the world from `seq-world`, and the ground of its zones: buildings, rock, water, fords and bridges, forest, roads and tunnel floor, and the patrol paths of mobs; and (D55) the server's own event names, quest steps, speed, mana, markers and zone ground, owed pay, and a duo in turn-based play; and (M8 cleanup, D56) no companion UI with the default setting, directing your second character from one session, the zone map without patrol paths, markers by their display names, owed pay listed at login, and a duo through the Warren; and (M9, D58) playing together: chat, parties across accounts, who and friends, and a Warren room cleared by two accounts.

- `model/`: a pure Go type built only from protocol messages (snapshot, events, intent results, zone map): you, entities in view, drops, encounters, the map, a bounded event log that keeps each event's reason, and the status of recent intents. It tracks the per-character stream number and takes each event exactly once (D39). `Text()` prints it as plain text.
- `client/`: a `Session` on the agent toolkit's connection (`agentkit` from the `seq` module, which owns the WebSocket, the login and the resume); its raw-message hook records each message and feeds the model. One command per intent (`Move`, `Stop`, `Target`, `Attack`, `Loot`, `UseDoor`, `Release`, `UseAbility`, `SetSlot`, `Logout`), and `Connect`, `CreateCharacter` and `Select` for a new character with `Result` matching each `intent_result`. There is no transport of the TUI's own.
- `sessionlog/`: the player's own log of every session (D64). The program records each session by default, under its play-session id, in a directory of its own under the data dir (`~/Library/Application Support/seq-tui/sessions` on macOS, `$XDG_DATA_HOME/seq-tui/sessions` on Linux, or `-log-dir`): `record.jsonl` (every message the server sent, the format of `-record` and of `seq-mcp`'s log) and `session.json` (the id, server without user info or query, character, zone, versions, start and end). One recording is capped at 32 MB and the directory at 256 MB (the oldest sessions go first); the credential is scrubbed from everything written. `-no-record` turns it off. Ctrl+L opens the sessions view (in play, and on the character creation screen): the recordings on this machine (Enter replays one in place, with Left and Right to step; E exports it to `exports/` beside the sessions) and the server's own list of the account's sessions (Enter reads the server's log as lines; E exports it). `-replay` takes a file, a session directory or a session id.
- `record/`: records a session's incoming messages as a log (one protobuf-JSON message per line) and reads it back.
- `testdata/logs/`: recorded logs the tests replay into the model with no backend. Tests only read them. `fight_loot`, `kill_steal`, `crossing` and `drop_resume` are built by hand (`go run ./cmd/mkfixtures testdata/logs`); `play`, `release`, `door`, `solo_trial`, `quick_turns`, `duo_turns`, `wizard`, `casts`, `warrior`, `levelup`, `turn_cast` and the `companion_*` and `duo_guard` logs are the sessions of the client scenarios, recorded by them (`make fixtures`). A scenario writes `# mark NAME` lines into its log at the points it names; the render tests and `model` tests replay a log up to a mark (`file@mark`), so a regenerated log moves nothing.
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

## Playing together (M9, D58)

Chat, parties, who and friends are the server's *social* requests, not intents (`client.Session.Social` and its wrappers on the toolkit's connection): each is answered by one result, with the refusal's reason when it is refused, and the status line says it in words. The model keeps what the server tells (`model.Social`: the chat scrollback of 500 lines, the party, the invitation waiting, the characters the who list named, the friends list); chat is not an event, so a resume does not replay it.

**/** opens the command line (the slash is in it already; Backspace takes it away to say a plain line). Enter sends, Esc cancels. Text with no command is a **say**, heard by the characters within 40 m.

| command | does |
| --- | --- |
| `/say TEXT`, `/p TEXT`, `/tell NAME TEXT`, `/r TEXT` | say, party chat, tell one online character, answer the last tell |
| `/mute NAME`, `/unmute NAME`, `/mute` | hide a character's say and tell from you; the list |
| `/invite NAME`, `/accept`, `/decline`, `/leave`, `/kick NAME`, `/party` | parties; the leader invites and kicks |
| `/who zone=NAME class=ID 3-7`, `/friends`, `/friend add NAME`, `/friend remove NAME`, `/anon on` | the who list (every filter optional), the friends list, hiding your zone |
| `/chat` | the Social screen on the scrollback |

A line of chat is colour-coded and tagged by channel in the **Chat** panel (say plain, party cyan, tell yellow; each line also says `[say]`, `[party]` or `[tell]`), which takes the right part of the log's row once anything has been said. The **Social screen** (`/chat`, `/who`, `/friends`) has the tabs Chat (Up / Down, PgUp / PgDn, End scroll the scrollback), Who and Friends (Up / Down choose, **I** invites, **F** makes a friend, **D** removes one, **R** asks again; Tab switches, Esc closes). A party invitation is a prompt across the top: **Y** accepts it, **N** declines it. The **Party** panel lists the other members (`*` leads) with health, mana, level and zone; a member out of view is shown with the zone and level the who list gave. A friend logging in or out is a line in the log. Any member can open an instance with **I**; the others see `instance opened for your party` and enter at the door, and in a turn-based one the Round panel marks each member's plan as the server tells it. The last page of the help (`?`) is on all of this.

## Danger and legibility (D75, D76)

**Danger is never hidden.** The traders, talk, board and contracts screens are panels in the map's area (on a window wide enough, 140 columns or so), with the You, Target and Nearby panels beside them; on a narrow one they take the width but keep a line of vitals (your health, your target, what attacks you) under them. Every other screen keeps that line too. When a mob attacks you (`self.in_combat`, `attackers`, and the `attack_began` event) the screen's border turns red and says so, the status line names the attacker and your health ("⚠ Meadow Wolf is attacking you — HP 113/130 — Esc to fight"), a hard or deadly mob is called out as DANGER (on the map as well), and the You panel says "in combat". Nothing closes by itself. The server offers the attacker as your target, so leaving the screen and pressing F fights back. `-bell` also rings the terminal's bell at each attack (off by default).

**A window stays with its person.** A traders', talk or board screen holds the NPC's id, not a place in a list sorted by distance, so a guard walking by changes nothing; it closes only when you are out of the NPC's reach, with a line on the status line saying so.

**Legibility.** Each role has a glyph: `T` trainer, `$` vendor, `B` banker, `H` binder, `g` guard, `Q` quest person, `&` board, `D` door. The server's objective hints draw the marks: a gold `!` for something new, a gold `?` for a hand-in that is ready, a grey `?` for a quest in progress, a gold board, gold quest mobs, and a gold `D` for the next door; an arrow at the edge of the map (`◄ Gorran 23m`) points to an objective that is off it, and the status line names it with its direction and distance. Titles ("Gorran Stoutarm <Warrior Trainer>") and the place name come from the server. Nearby lists what matters to the quest first, with tags (`[step 2]`, `[2/3]`, `[new]`, `[next door]`). `░` is the edge of the zone. The legend is on the second page of the help and on the zone map (`M`). A line above the status line says which key does what next to a door, a person, a drop or a mob.

**A readable log.** The log shows the server's own words for every event (`Event.text`, `Denied.text`), never an id or a coordinate, and hides regeneration ticks and movement (V shows them). A level gained is a banner across the top for a short while (the new level, health and mana gained, what can be trained and where); a contract handed in shows what it paid in the board panel; a finished contract says where to hand it in.

## The mouse and the character sidebar (D77, D78)

**The mouse is a convenience; every action has a key** and the game is fully playable without it (`-no-mouse` leaves the terminal's own mouse alone). The program asks the terminal for clicks, the wheel and motion (xterm modes 1003 and 1006, SGR reports; Bubble Tea's `MouseModeAllMotion`). Shift+click is never used: terminals keep it for selecting text.

| Mouse | Key |
| --- | --- |
| Click the ground (on the map or the large map): the server plans the route and you walk there; routes stay inside the zone | arrows or WASD; on the large map the arrows move a cursor and Enter walks there |
| First click on a mob or a person: selects it (a mob becomes your target) and opens a small card: name, title, level and difficulty, what they offer. A first click never attacks | Tab (target); the Target panel and Nearby say the same |
| Second click on the selected mob attacks; on a person it talks (a person out of reach: it walks you up to them) | F attacks, G talks |
| Click loot: loot-all (loot out of reach: it walks you there, and a second click takes it) | L |
| Hover: a name-and-title label by the pointer (a drop also says how it compares with what you wear) | Nearby lists the same |
| Hold Alt over the map: everything on it is labelled | Ctrl+A turns the labels on and off, for a terminal that sends no Alt with the mouse |
| Scroll wheel: zoom the map (and the large map) | + and - |
| Right-click, or Esc: let go of the selection | Esc |

**M opens the large map**: the whole zone you have seen at full size, the places' names (streets, squares, buildings, gates, fields), the quest marks, the legend, and the vitals strip under it. Click a spot to walk there. The wheel (or + and -) zooms in on your character.

**The character sidebar (D78).** Beside a vendor, trainer, bank, contract board or conversation, a sidebar keeps your character in view under the You panel: **Gear** (what you wear, with the slots in words: "Main hand", "Ring 1"), **Bags** (what you carry, identical things stacked: "Wolf Pelt ×2", with the slots used) and **Stats** (the server's character sheet: health, mana, armour and what it cuts, accuracy and the chance to hit, what your swing does now, bare hands too — "Fists: 7–13 / 2.0s" — and each attribute with what gear adds). **C** turns to the next tab, a click on a tab picks it, PgUp / PgDn or the wheel scroll. In a window too narrow for it (80 columns), the sidebar is a drawer: **C** swaps it in for the screen, **C** or Esc swaps it back, and the vendor's window is still there.

**Every item says how it compares with what you wear in its slot** (the server works it out, D78): `▲` upgrade, `▼` downgrade, `=` about the same, `x` your class or level cannot use it (the line says why, and it is greyed out). The mark is on vendor lists, bags, the bank, loot (Nearby, hover), and contract rewards ("Pays 21 XP and Tin Ring ▲"). The highlighted item's card says what it is, what it adds, what it costs or what a vendor pays, and how it compares ("upgrade over an empty main hand: +1.5 damage/s"). What you cannot afford is greyed out with how far short you are ("120, short 90"). At a vendor **J** sells all junk in one go (things with no gear slot that no contract of yours wants), **A** sells the whole stack under the cursor, Enter one. **L** loots everything within reach and the log shows one line: "You took Worn Cap, Scuttler Blade, 5 crowns". The first time you hold an item that is "tradable once", the log explains it.

**Terminals.** Reports are the standard xterm ones, so a terminal with SGR mouse reporting (mode 1006) works. Holding Alt only labels everything when the terminal sends the modifier with the mouse report (and, for hover, with motion); where it does not, or the window system keeps Option or Alt for itself, **Ctrl+A** is the toggle. None of iTerm2, Terminal.app or Ghostty was tried by hand in the mission that built this: no GUI application may be launched there. What the scenarios prove is the program's side: they send the same SGR bytes (press, release, motion, wheel, Alt) to the real running program. If a terminal ignores the mouse, or reports it badly, `-no-mouse` turns it off and the keys do everything.

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

## The world and the ground (D53, D54)

The world is not in this repository. The zones, people, items and contracts of Thornwick, the Thornwick Downs and the Hollow Warren are `seq-world`'s (a pinned Go module in `go.mod`), and the scenarios load them from it through the test server (`testserver.Options{Packs, Zones}`); `make play` serves the same packs with `seqd -packs`. `SEQ_WORLD_PACKS=dir` points scenarios and `make play` at a checkout of `seq-world` under work instead. Move the pin with `go get github.com/GeneStevens/seq-world@<commit>`, then `make fixtures golden` and review the diff.

The map draws the ground under everything else, one glyph per kind, so it reads on a monochrome terminal and at 80x24; colour only repeats it:

| glyph | ground | glyph | ground |
| --- | --- | --- | --- |
| `#` | building (also a wall) | `"` | forest (slows you) |
| `^` | rock | `·` | road (speeds you) |
| `≈` | water | `.` | a dungeon's floor |
| `:` | ford | | |
| `=` | bridge | | |

In a dungeon (a zone with floor, like the Warren) only the floor is drawn, with the rock that touches it as walls. The **status line** gives the server's word on the character's speed (D55, `self.ground` and `speed_pct`), as in `forest: slowed to 75%` or `road: faster, 125%`. **M** draws the whole zone. The paths that camps and guards walk are not told (D56): the map shows them where they are, and a guard on its patrol is streamed (`npc_moved`) like any mover.

**The ground is the zone map's (D55)**: its `areas` are drawn as sent, the model builds them from the map message (`model.TerrainOf`), and nothing is read from the packs by the client any more. The server states the ground and speed only in a snapshot, and a move does not refresh them, so the status line uses its word while the map's areas still put the character on the ground it named; once it has walked onto other ground it names that ground alone, until the next snapshot says the speed there. A guard is an NPC the server streams as it walks (D56), and the screen draws it where it is.

## People, contracts, events and the map

**G** talks to the nearest person you have seen (the server decides whether they are within 5 m). Their lines and what they offer are on the **Talking** screen, with the options as the server sent them; an option opens the screen it leads to: buying and selling, training, binding and banking open that trader's window (the existing N screens), and taking or handing in a contract opens the **contract board**. A person with nothing to offer yet (the Watch Captain) says so. Up / Down or a number choose, Enter does it, Esc closes.

The **contract board** lists what it offers you now (name, kind, levels; Tab for the contracts you hold) with the description and the pay. Enter takes an offered contract, or hands in a held one; X gives a held one up. The client sends the intent and the server refuses with its reason (`contract_limit`, `contract_held`, `contract_not_done`, `out_of_range`). **K** opens the **Contracts** screen (and a Contracts panel appears in the right column when there is room): each contract held with its progress in words (kills, items carried, or "go to the marker" and then "marker reached"), where a scout's marker is, and the pay; X gives the selected one up. The server describes a held contract from the zone the character is in, and the Downs have no board, so the client keeps the name, text and pay it saw on the board and fills a held contract in from them.

A **world event** is announced to the zone with its name (D55; the client makes none from its id): a banner row across the top says what is on and, for thirty seconds after it ends, how it ended; the log has every announcement, leak and payout. A **roaming rare** in view is marked `r` on the map, `RARE` in Nearby and the Target panel, and a log line says it has appeared. The server marks a rare (`EntityState.rare`, D52), so the client keeps no list of names.

The map scrolls: it is always centred on the character at a fixed scale, and its title says where in the zone that is. `&` marks a contract board, `?` someone who only talks, `$` the other traders, `%` the marker of a scout contract you hold, `D` a door, and `c` a camp (the zone map's camps: where its mobs are found). **Every marker of the zone map is drawn** (D55): `D` for a zone line or an instance entrance, `%` for the scout marker a held contract names and `;` for the other scout markers, and for a world event `!` where its raiders appear and `G` where they are heading. NPCs stream as you walk (`npc_appeared`, `npc_left`), and the client keeps the ones it has seen, in each space, and draws them wherever they stand; a trader beyond its own reach (the NPC's `reach_mm`) is marked `too far` and greyed in the traders' list. Nothing is guessed (D52): the Peril fee, an item's tier, binding and stats (from the drop and reward events), a trainer's ability names and descriptions, the bind fee, a held contract's name, text and reward and the round a cast lands in are all read from the protocol. **M** shows the whole zone in the panel, with what has been seen of it.

**Owed pay (D55).** A world event pays a character that left the zone before it ended at its next join, and a refund owed to an opener who had left is paid at its next join too. The server sends no event for it: a stream starts from the snapshot, which already shows the join, so the reward events of the join's tick are not delivered. The client therefore says it from what it can see: a snapshot after a door shows more XP or crowns than the character had, and the log says `owed pay arrived with the join: 60 XP and 30 crowns`. A fresh login has nothing earlier to compare, so it says nothing; a reward event that arrives for an event whose end this program was not told of (or a refund no failed opening answers for) is told as owed as well.

**Deaths (D55).** A death in an instance wakes the character at the instance's entrance, inside the instance; the map, the Instance panel and the Round panel follow the new snapshot, and nothing in the client decides where the character wakes.

## Classes and combat

`seq-tui -new` (or `SEQ_NEW=1 make play`) opens the **creation screen**: type a name (3 to 16 letters; the server makes its id the name in lower case and refuses a taken one with its reason), Tab or Down moves to the **class menu the server sent with its welcome** (the client has no class text of its own: each class's name, roles, description and the kit it starts with, free, all come from it; Up / Down or the number choose), Enter makes the character and plays it. A refusal (`bad_name`, `name_taken`, `unknown_class`, `too_many_characters`) is shown in words on the screen, which stays.

A character with abilities has an **ability bar** across the screen: eight slots, each with the ability, its mana cost on a wide screen, and one word for whether it can be used now: `ready`, `busy` (a cast is running), a cooldown's seconds left, `gcd` (the global cooldown), `mana`, `tgt?` (an enemy ability and no target) or `range`. The words are a hint from the model's own numbers; the key always sends the intent and the server decides, and its refusal (`no_mana`, `on_cooldown`, `out_of_range`, ...) is on the status line in words. The bar's last row lists the effects on you and on your target with the time left (a buff or damage over time counts rules ticks, so its time is marked `~`; a stun, root or snare says Stunned, Rooted, Snared). The You panel has the class, an HP bar, a mana bar for casters, an XP bar with what is left to the next level, and the **cast bar**: the ability and the time left. A cast that does not land (you moved, damage broke it, you were stunned, the target is gone) is a red `!` line in the event log and in the You panel until the next cast begins. A level-up is a `+` line in the log that names the new ability and its key. Mana changes are in the log under **V** only; the bar shows them.

**Quest chains and the help (D62, D66, D68, D69; TUI-024).** A new character in Thornwick is offered **First Steps** (a chain; the server's text). While the character is on a chain the **Quest chain** panel and the status line (`First Steps 2/4`) show it and its current step, and the log says in plain words when it is offered, a step is done (with what it paid), it is finished or dismissed. **Ctrl+K** (or `/quest`) opens the chain's screen; **X** there (or `/dismiss`) asks one question, and **Y** gives the chain up for good (the server never offers it again). Nothing walks the character anywhere or reminds it (D57). A fight that stops a walk (D66: attacking, a hostile ability, or a mob engaging) is a log line, "Your walk stopped"; an attack or ability with nothing targeted goes to what is attacking the character, and the status line names the mob the server picked. The zone map's standing NPCs (D69) are on the map and in Nearby by name before they are seen. **`?` or F1** opens the **help** from every view (Esc closes it, `?` turns the page, Left and Right turn it either way, Up and Down scroll): five pages, on moving and doors, on talking, trainers and contracts, on quest chains and fighting, on logs and replay, a page of every key, and the page on playing together. The page of keys is drawn from `screen.Bindings`, and `app/bindings_internal_test.go` holds the key handler to that table both ways, so the help cannot drift from the keys.

**B** opens the **slot screen** (out of combat; the server refuses it in combat with `in_combat`): Up / Down or 1 to 8 choose a slot, Enter picks an ability from the list of what the character knows, Backspace empties the slot, Esc closes it. In a **turn-based** instance a slot changes with the plan (D50): the same keys add a step, the slot screen marks it `(in the plan)`, Enter on the main screen submits it, and the slot holds when the round opens (it is allowed in a fight there). A cast that will land in a later round says so (`lands in round N`) in the You panel, the Round panel and the log.

## Items, traders and trade

The screens below read only what the snapshot and the events give (the snapshot carries the item catalogue, the traders in view and the trade window); the server decides every action and its refusal is on the status line in words.

- **C** is the character view, the sidebar's Gear, Bags and Stats as a panel of its own with the line of vitals under it: the ten gear slots, the items carried (each with its mark `▲ ▼ = x`, its tier, whether it is bound — `[bound]`; Rare and Named are bound on pickup, a traded item is marked `!` and bound to its owner — and what it adds) and the server's character sheet. It opens on the bags when they hold anything. Tab (or 1 to 3) turns to a tab, the arrows choose, Enter wears the item in the bags or takes off the gear under the cursor (out of combat), Esc or C closes. The card under the list says how the item compares with what is worn.
- The **You** panel shows crowns at the right of its first line. Drops that the server marks as not yours to take (a Rare drop you are not entitled to) are `~` on the map and `not lootable` in Nearby, and **L** says so without asking the server.
- **N** lists the traders in view (`$` on the map): Enter opens a **vendor** (the stock on the left, what you carry on the right; Enter buys one or sells one), a **trainer** (the abilities of your class with level and cost; Enter learns the one under the cursor into the first empty slot), a **binder** (Enter sets the bind point) or the **bank** (Tab: deposit or withdraw).
- **T** is the trade window. With no window it lists the characters in view and in your group; Enter asks one, and the other presses T and Enter to agree. Then each side puts items in (Enter on what you carry), sets crowns (**C**, digits, Enter), takes things back (Tab to the offer, Enter), and **Y** confirms; any change clears both confirmations; **X** cancels. Bound items are refused by the server with `bound`.
- The instance menu scrolls, and marks the heroic (Peril) presets: the server takes an entry fee in crowns when they open (the protocol does not say how much; the log does, once it is paid, and a refund shows there too).

## Companions

**Shelved (D56).** The house companion is off in the server by default (`seqd -companions` turns it on): the snapshot then carries none, the Captain offers no quest, and the client shows nothing of it: no Companion or Quest panel, no companion lines in the help, no recruit option in the Captain's talk, and `P H Y U J Z` say there is no helper unless a second character is in play. Everything below is what the client does when the server has a companion.

A **house companion** is a server-run recruit of the Thornwick Watch (SEQ-049, D35, D43, D46): one for each account, it has your level, fights beside you at about 70 % of a player's strength, takes a party slot, and earns nothing. The client holds none of that; it shows what the server says and sends intents.

**The quest.** Talk to Captain Rhosyn Hartley (**G** beside her): her board offers the first step of her quest (a scout of the goblin lookout), the second (a hunt of five goblins) only once the first is handed in. A **Quest** panel lists the steps as seen (`[ ]` offered, `[>]` taken with its progress, `[!]` ready to hand in, `[x]` done) and says what to do next. The server names the quest and the step on a contract and on an offer (D55: `quest`, `step`), so the panel lists the steps by their number, and it knows only the steps it has been told of (offered or held) since the program started. The last hand-in unlocks the companion; the Captain's option **Choose your recruit's role** opens a screen where Up / Down or 1 to 4 choose tank, healer, melee or caster and Enter sends it (she lets you change it whenever you like).

**The Companion panel** (right column, once a role is chosen) shows its name and level, its hit points, its stance and what it is doing (its order, the target it is on, or what its stance does). Its mana (a healer's or a caster's) is shown as an MP bar, from the group's entry in the snapshot (D55); a snapshot carries it, and it changes between snapshots only if the server sends a resource event for the companion (the docs say others' mana changes with a snapshot). On a small screen the panel takes the place of the Target panel.

| key | does |
| --- | --- |
| **P** | call the companion to your side, or send it away (it then comes by itself whenever you join a space, until sent away) |
| **H** | the next stance: follow, assist, guard, heal, passive |
| **Y** | attack my target (an order: it overrides the stance) |
| **U** | come here (walk to where you stand, and stand) |
| **J** | stop (stand and fight nothing until told) |
| **Z** | drop the order, back to the stance |
| **.** | with a second character as well, point these keys at the other helper |

It is never independent (D35): it goes when your connection drops and is back when you are. In a **turn-based** instance the same keys add steps to the plan (a stance and an order, one of each; the plan lists them as `companion: stance assist`, `companion: attack Plain Scuttler`) and the companion does only what the plan says.

**A second character on a stance (D42).** `seq-tui -duo bram` plays the account's second character from the same program, with the toolkit's built-in stances (`agentkit.Stance`): it logs in on a connection of its own when **P** is pressed, and a **Party** panel shows it with the same keys. Its orders are the same words (attack my target, come here, stop); the driver is `client.Duo`, which reads the character's own replica about twice a second of game time and sends what the stance says. **E** at a door sends the second character through it as well (the server decides whether it is within reach; its refusal is on the status line). In a **turn-based** instance (D55) the stance plays by plan: each round that opens, `client.Duo` sends the plan its stance makes of its own replica (`agentkit.PlanSteps` of what it would send: at most one move and one action). With an agent credential for it (`SEQ_DUO_AGENT_SECRET`) the plan is a **stand-in** (D10): the server uses it at the deadline for a character with no plan of its own, it does not count as a submission, so the round runs at its deadline, and the Round panel shows what it planned (`bram plans: target Plain Scuttler, attack`) and then, when the round has run, `bram: stand-in plan`. Without that credential (a session token may not send a stand-in) the plan is sent as the character's own, and counts as a submission. **O** logs both characters out.

**Directing the second character (D56, Pillar 10).** **X** switches the keys and the screen to the account's other character, once it is in play (**P**): the screen is then its own (its hit points, abilities, round and plan), the status line says `directing bram`, and every key is its key, sent by this session as a command for that character (`Intent.character`; the server plays it as that character's and answers this session). Arrow keys move it, **Tab** targets, **1** to **8** cast its abilities, **F** attacks, **E** takes its door. Its stance waits while it is directed; **Z** hands it back to its stance (**H** changes the stance and does the same), **X** gives the session back to the first character. In a **turn-based** round each character has a plan of its own: the Round panel shows `bram's plan (3/16)` for the one directed, **Enter** submits it for that character (it counts as its own plan, so a round can run on this one session's say-so), and **X** shows the first character's plan, kept apart. The server refuses a directed command for any character that is not the account's, or not in this space (`not_here`), and says so on the status line. Directing is not presence for either character (D31). The stance follows its leader through a door on its own (D56): if the second character is too far to be sent through the door with **E**, its stance walks to the door the leader left by and takes it.

The scenarios (`scenario/companion_test.go`) press their own keys: the first step of the quest from the Captain; the scout of the lookout; the last step, the choice of role and calling the recruit; a camp fought with the companion on assist, then overridden with J and sent back with Y; the five stances; a drop and a fresh login with the companion; a turn-based round with a companion step in the plan; a second character on guard. The M8 cleanup scenarios (`scenario/m8_test.go`): no companion UI with the default setting; directing the second character (move, target, cast, then follow, then back); a directed plan in a turn-based round; owed pay listed at login; a guard that walks on the screen and no patrol path; a duo through the Hollow Warren (in by stance, a room cleared, out by direct orders). The D55 scenarios (`scenario/d55_test.go`): a duo on a stance planning each round of a turn-based instance by stand-in; the server's speed in a forest and out of it; a death in an instance waking at its entrance; owed pay arriving with a join; the companion's and the party's mana; every marker of a zone on the map. A scenario cannot hold the whole quest, since a contract counts in its own zone, so the quest is played in the three pieces that its two zones allow. `World.LoginWithDuo` (and `LoginWithDuoAgent`, with an agent credential for the second character) gives a player a second character that the scenario drives once each time the player syncs, so that it is deterministic. No scenario sends the second character away: logging a character out over the test server can stall `Advance` for five seconds when the session ends while the server settles.

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

The economy scenarios (`scenario/economy_test.go`): kill the trophy scuttler, loot it and wear the loot while a bystander sees the Rare drop as not lootable; buy, sell, train, bind and bank at the traders; a trade between two characters with a bound item refused; open a Peril instance and pay the fee. The Thornwick scenarios (`scenario/town_test.go`) run on seq's own test server with the Thornwick zones; they include walking up to a trainer (it appears by event), dying in the Downs and waking in Thornwick, a scout contract's marker and the Downs' camps from the zone map, and Old Grimtusk by its flag.

The Thornwick scenarios (`scenario/town_test.go`) run on `scenario.StartTown`, a test server built like seq's `testserver` (the same packages and manual clock) that runs Thornwick and the Thornwick Downs, which `testserver` does not: create a character, start in the square, talk to the board and take a Hunt (and the limit of three, and giving one up); walk out of the east gate, kill the wolves, come back and hand it in; buy from a vendor and train an ability; a Scout contract to a marker; the Harvest Raid announced and joined; Old Grimtusk coming into view.

The older scenarios (`scenario/flows_test.go`): login, move, target, attack, kill and loot; die, release and respawn; the door across and back; solo-trial from the menu to the hand-back; quick-turns round by round (plans edited with Backspace and Esc, the receipt, a missed deadline, a dropped and resumed connection, a fresh login into the open round, clear); a duo on one account entering together in the gathering round.

Speed gates (D49), enforced in the test code: each scenario under 2 s, the whole suite under 15 s (`scenario.Main` in `TestMain`). `SEQ_SCENARIO_SLOW=3s go test ./scenario` makes each scenario wait 3 s at its end and shows the gate failing. The suite's gate counts the whole run, so it means something with `-count=1`.

`make fixtures` runs the scenarios with `SEQ_FIXTURES` set, which makes the scenarios that declare a fixture write their recording to `testdata/logs`, and rebuilds the hand-built logs. Run `make golden` after it, and review both diffs. (`make fixtures` also rewrites the scenario goldens as it runs, so a screen that changed does not stop the recording half way.)

## Verify

```
make verify
```

Runs a `gofmt` check, `go mod download github.com/GeneStevens/seq-world`, `go vet ./...` and `go test ./...` (which includes the scenarios), then prints the elapsed time. It must stay well under 60 seconds.

The download is a step of its own because the scenarios load seq-world's packs from the module cache (`go list -m -f '{{.Dir}}'`), and nothing in the code imports that module, so `go build` and `go vet` never fetch it. On a fresh machine, or after the pin moves, the first `make verify` needs access to the private `github.com/GeneStevens/*` modules (`GOPRIVATE` and the ssh rewrite, as above); a missing module fails the step with the module's name instead of failing every scenario with `seq-world is not in the module cache`.

## History

The pre-reset code is preserved under the annotated tag `pre-reset` (commit `9d04d89`). `main` was restarted fresh from it; history is kept. See `PROVENANCE.md` for what is excluded.

The map view uses the technique of the old rasterizer (`mapgeo.go` at `pre-reset`): line-to-cell rasterization in a player-centred, snapped viewport. It is written again here against the protocol's zone map; none of its code or EverQuest-derived data is carried (see `PROVENANCE.md`).

## Colour, and `NO_COLOR` (TUI-029)

A panel's title is bold in one accent colour (cyan); a section inside a panel ("For sale", "You hold", "Gear worn") is bold; a help line is dim with its keys highlighted; READY is green and a refusal is red. Gold is the quest's and marks nothing else. The colours are the terminal's own 16, so they follow a dark or a light theme. With `NO_COLOR` set (to anything), no colour is sent at all, and bold, dim and reverse are kept: every distinction is in the words and glyphs too. The Events panel opens short (`Events (short · v: all)`): movement, regeneration and people coming into or going out of view are left out until V shows them (`Events (all · v: short)`).
