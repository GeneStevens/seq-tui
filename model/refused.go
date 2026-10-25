package model

// refusedWithinMs is how close two identical refusals are for the log to keep one
// of them: the intent's result and the denial event say the same thing.
const refusedWithinMs = 2000

// noteRefused puts a refused intent in the log in the server's plain words, where
// the player's eye is on the Events panel. A denial event that said the same a
// moment ago (or says it next) is the same line.
func (m *Model) noteRefused(text string) {
	if text == "" {
		return
	}
	m.addLog(LogEntry{Seq: m.last, GameTimeMs: m.GameTimeMs, Kind: "refused", Text: text, Actor: m.Self.ID})
}

// Refused says the log entry is a refusal the player is shown in red: a refused
// intent, a denial event, or the server's word that a game master muted the character.
func (e LogEntry) Refused() bool {
	return e.Kind == "refused" || e.Kind == "denied" || e.Kind == "notice_muted"
}

// repeatsRefusal says e is a refusal with the words of one logged within the last
// refusedWithinMs.
func (m *Model) repeatsRefusal(e LogEntry) bool {
	if !e.Refused() {
		return false
	}
	for i := len(m.Log) - 1; i >= 0; i-- {
		o := m.Log[i]
		if o.GameTimeMs < e.GameTimeMs-refusedWithinMs {
			break
		}
		if o.Refused() && o.Text == e.Text {
			return true
		}
	}
	return false
}

// DoorUsable says whether the server lets the character use the door and, when it
// does not, why in its plain words. A door the server has said nothing of is
// taken as usable: the server decides when it is pressed.
func (m *Model) DoorUsable(id string) (usable bool, why string) {
	d := m.Doors[id]
	if d == nil || d.GetUsable() {
		return true, ""
	}
	return false, d.GetWhy()
}
