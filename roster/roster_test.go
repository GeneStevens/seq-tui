package roster_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/roster"
)

func TestNoteListForgetAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "roster.json")
	r, err := roster.Load(path)
	if err != nil || len(r.List()) != 0 {
		t.Fatalf("a missing file: %v, %v", r, err)
	}
	t0 := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	r.Note(roster.Entry{ID: "c1", Name: "Zehen", Class: "warrior", Level: 3, Zone: "Thornwick", Played: t0})
	r.Note(roster.Entry{ID: "c2", Name: "Mira", Class: "wizard", Level: 1, Zone: "Thornwick", Played: t0.Add(time.Hour)})
	r.Note(roster.Entry{ID: "c1", Name: "Zehen", Class: "warrior", Level: 4, Zone: "Thornwick", Played: t0.Add(2 * time.Hour)})
	r.Note(roster.Entry{Name: "nobody"})
	l := r.List()
	if len(l) != 2 || l[0].ID != "c1" || l[0].Level != 4 || l[1].ID != "c2" {
		t.Fatalf("list = %+v", l)
	}
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	back, err := roster.Load(path)
	if err != nil || len(back.List()) != 2 || !back.List()[0].Played.Equal(t0.Add(2*time.Hour)) {
		t.Fatalf("reloaded: %+v, %v", back, err)
	}
	back.Forget("c1")
	if l := back.List(); len(l) != 1 || l[0].ID != "c2" {
		t.Fatalf("after forget: %+v", l)
	}
}

func TestBadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := roster.Load(path); err == nil {
		t.Fatal("a corrupt roster loaded")
	}
}
