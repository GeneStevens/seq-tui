package main

// rosterFocus holds purely local, non-authoritative focus state for the
// encounter roster. It has no gameplay meaning, triggers no backend writes,
// and exists only to let the user place visual attention on a roster entry.
type rosterFocus struct {
	index int // index into the flat roster entry list; -1 means no focus
}

// rosterEntry is one entry in the flat roster list used for focus navigation.
// The kind prefix ("pc" or "mb") is stored alongside the ID for display.
type rosterEntry struct {
	kind string // "pc" or "mb"
	id   string // backend-owned ID, displayed as-is
}

// buildRosterEntries returns the flat, deterministic list of roster entries
// for an encounter summary. Players first, then mobs, preserving backend order.
// Returns nil if encounter is nil or has no participants.
func buildRosterEntries(enc *encounterSummary) []rosterEntry {
	if enc == nil {
		return nil
	}
	total := len(enc.PlayerIDs) + len(enc.MobIDs)
	if total == 0 {
		return nil
	}
	entries := make([]rosterEntry, 0, total)
	for _, pid := range enc.PlayerIDs {
		entries = append(entries, rosterEntry{kind: "pc", id: pid})
	}
	for _, mid := range enc.MobIDs {
		entries = append(entries, rosterEntry{kind: "mb", id: mid})
	}
	return entries
}

// reconcileFocus adjusts the focus index to remain valid against the current
// roster entries. If the previously focused entry still exists, focus stays on
// it by ID. Otherwise focus is clamped to the last entry, or cleared to -1 if
// the roster is empty.
func reconcileFocus(f rosterFocus, oldEntries, newEntries []rosterEntry) rosterFocus {
	if len(newEntries) == 0 {
		return rosterFocus{index: -1}
	}
	if f.index < 0 {
		// No prior focus — stay unfocused
		return rosterFocus{index: -1}
	}

	// Try to find the previously focused entry by identity in the new list
	if f.index < len(oldEntries) {
		prev := oldEntries[f.index]
		for i, e := range newEntries {
			if e.kind == prev.kind && e.id == prev.id {
				return rosterFocus{index: i}
			}
		}
	}

	// Previously focused entry disappeared — clamp to last entry
	return rosterFocus{index: len(newEntries) - 1}
}

// moveFocusDown moves focus down within the roster. If unfocused, focuses the
// first entry. Clamps at the end of the list.
func moveFocusDown(f rosterFocus, entryCount int) rosterFocus {
	if entryCount == 0 {
		return rosterFocus{index: -1}
	}
	if f.index < 0 {
		return rosterFocus{index: 0}
	}
	next := f.index + 1
	if next >= entryCount {
		next = entryCount - 1
	}
	return rosterFocus{index: next}
}

// moveFocusUp moves focus up within the roster. If unfocused, focuses the
// last entry. Clamps at the start of the list.
func moveFocusUp(f rosterFocus, entryCount int) rosterFocus {
	if entryCount == 0 {
		return rosterFocus{index: -1}
	}
	if f.index < 0 {
		return rosterFocus{index: entryCount - 1}
	}
	next := f.index - 1
	if next < 0 {
		next = 0
	}
	return rosterFocus{index: next}
}
