// Package roster is the TUI's own list of the characters it has played on a
// server: the picker's source. The protocol has no request for an account's
// characters, so the program notes each one it makes or plays (id, name, class,
// level, zone, when) in a small JSON file and lists them from there.
package roster

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Entry is one character as last seen playing.
type Entry struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Class  string    `json:"class,omitempty"`
	Level  int64     `json:"level,omitempty"`
	Zone   string    `json:"zone,omitempty"`
	Played time.Time `json:"played"`
}

// Roster is the characters, in no particular order; List sorts them.
type Roster struct {
	Entries []Entry `json:"characters"`
}

// Load reads the roster at path. A file that is not there is an empty roster;
// one that cannot be read as a roster is an error.
func Load(path string) (*Roster, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Roster{}, nil
	}
	if err != nil {
		return nil, err
	}
	r := &Roster{}
	if err := json.Unmarshal(b, r); err != nil {
		return nil, err
	}
	return r, nil
}

// Save writes the roster to path (and its directory), through a temporary file.
func (r *Roster) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Note records a character as seen now, replacing its earlier entry.
func (r *Roster) Note(e Entry) {
	if e.ID == "" {
		return
	}
	for i := range r.Entries {
		if r.Entries[i].ID == e.ID {
			r.Entries[i] = e
			return
		}
	}
	r.Entries = append(r.Entries, e)
}

// Forget drops a character the server no longer knows.
func (r *Roster) Forget(id string) {
	r.Entries = slices.DeleteFunc(r.Entries, func(e Entry) bool { return e.ID == id })
}

// List is the characters, the last played first (ties by name).
func (r *Roster) List() []Entry {
	out := slices.Clone(r.Entries)
	slices.SortStableFunc(out, func(a, b Entry) int {
		if c := b.Played.Compare(a.Played); c != 0 {
			return c
		}
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		}
		return 0
	})
	return out
}
