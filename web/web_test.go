package web_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq-tui/web"
)

// browserKey is a key a browser may act on itself: a Ctrl+letter (the page
// cannot see Ctrl+Shift or Alt chords as the game does) or a function key.
var browserKey = regexp.MustCompile(`^(ctrl\+[a-z]|f[0-9]{1,2})$`)

// A binding that clashes with a browser shortcut must be listed in keys.json,
// to be kept from the browser, or the game would lose it in the page.
func TestBindingsThatClashWithTheBrowserAreIntercepted(t *testing.T) {
	keys, err := web.InterceptedKeys()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range screen.HelpBindingKeys() {
		switch {
		case !browserKey.MatchString(k), k == "ctrl+c": // Ctrl+C reaches the game as ^C
			continue
		case slices.Contains(keys.Unreachable, k):
			t.Errorf("binding %q cannot be kept from the browser (it opens or closes a tab); rebind it", k)
		case !slices.Contains(keys.Intercept, k):
			t.Errorf("binding %q is a browser shortcut and is not in web/keys.json's intercept list", k)
		}
	}
}

func TestInterceptListIsWellFormed(t *testing.T) {
	keys, err := web.InterceptedKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys.Intercept) == 0 {
		t.Fatal("no keys intercepted")
	}
	for _, k := range append(slices.Clone(keys.Intercept), keys.Unreachable...) {
		if !browserKey.MatchString(k) {
			t.Errorf("%q is not a Ctrl+letter or a function key in the terminal's key names", k)
		}
	}
	for _, k := range keys.Intercept {
		if slices.Contains(keys.Unreachable, k) {
			t.Errorf("%q is both intercepted and unreachable", k)
		}
	}
	for _, want := range []string{"ctrl+l", "ctrl+k", "ctrl+r", "ctrl+u", "ctrl+a", "f1"} {
		if !slices.Contains(keys.Intercept, want) {
			t.Errorf("%q must be intercepted", want)
		}
	}
}
