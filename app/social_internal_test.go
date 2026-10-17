package app

import "testing"

func TestSplitLineReadsCommandsAndPlainText(t *testing.T) {
	for _, c := range []struct{ in, cmd, rest string }{
		{"hello there", "", "hello there"},
		{"/say hi", "say", "hi"},
		{"/TELL Bob  psst, hello ", "tell", "Bob  psst, hello"},
		{"/leave", "leave", ""},
		{"  /p all in", "p", "all in"},
	} {
		if cmd, rest := splitLine(c.in); cmd != c.cmd || rest != c.rest {
			t.Errorf("%q: %q %q, want %q %q", c.in, cmd, rest, c.cmd, c.rest)
		}
	}
}

func TestLineVerbs(t *testing.T) {
	for cmd, want := range map[string]string{"s": "say", "p": "party", "party": "party?", "t": "tell", "w": "tell", "r": "reply", "ignore": "mute",
		"inv": "invite", "kick": "kick", "who": "who", "friends": "friends", "friend": "friend", "dance": "?"} {
		if got := lineVerb(cmd); got != want {
			t.Errorf("/%s: %q, want %q", cmd, got, want)
		}
	}
}

func TestWhoFilters(t *testing.T) {
	zone, class, lo, hi, err := whoFilter("zone=thornwick class=cleric 3-7")
	if err != nil || zone != "thornwick" || class != "cleric" || lo != 3 || hi != 7 {
		t.Fatalf("%q %q %d %d %v", zone, class, lo, hi, err)
	}
	if _, _, lo, hi, err := whoFilter("5-"); err != nil || lo != 5 || hi != 0 {
		t.Fatalf("5-: %d %d %v", lo, hi, err)
	}
	if _, _, lo, hi, err := whoFilter("-9"); err != nil || lo != 0 || hi != 9 {
		t.Fatalf("-9: %d %d %v", lo, hi, err)
	}
	if _, _, _, _, err := whoFilter("nonsense"); err == nil {
		t.Fatal("a bare word is no filter")
	}
}
