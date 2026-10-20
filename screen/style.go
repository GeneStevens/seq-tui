package screen

import "strings"

// Styler decorates text for a terminal. Layout is always computed on plain
// text first and styled after, so a style never changes a width. Colour is
// only ever an extra: every distinction a style makes is also made by a glyph
// or a word (doc 01, accessibility).
//
// The hierarchy of a screen is made of three things, in this order: Accent for
// the title of a panel, Bold for a section header inside it, and Key for the key
// a help line names, over a Dim line. Green means ready or done, Red a refusal or
// danger; Yellow is the quest's gold and is used for its marks only.
type Styler struct {
	Bold, Dim, Reverse func(string) string
	Red, Green, Yellow func(string) string
	Cyan, Blue         func(string) string
	Accent, Key        func(string) string
}

func ident(s string) string { return s }

// Plain is the Styler that adds nothing: golden screens and replays use it.
func Plain() Styler {
	return Styler{Bold: ident, Dim: ident, Reverse: ident, Red: ident, Green: ident, Yellow: ident, Cyan: ident, Blue: ident, Accent: ident, Key: ident}
}

func sgr(code string) func(string) string {
	return func(s string) string {
		if s == "" {
			return s
		}
		return "\x1b[" + code + "m" + s + "\x1b[0m"
	}
}

// ANSI is the Styler for a terminal: SGR attributes and the 16 basic colours,
// which the terminal maps to its own palette, so that it reads on a dark
// terminal and on a light one. The accent is cyan; titles are bold in it and
// keys are plain in it.
func ANSI() Styler {
	return Styler{Bold: sgr("1"), Dim: sgr("2"), Reverse: sgr("7"), Red: sgr("31"), Green: sgr("32"), Yellow: sgr("33"), Cyan: sgr("36"), Blue: sgr("34"),
		Accent: sgr("1;36"), Key: sgr("36")}
}

// Mono is ANSI without colour, for NO_COLOR (no-color.org): the hierarchy is
// kept with bold, dim and reverse, which are not colours.
func Mono() Styler {
	st := ANSI()
	st.Red, st.Green, st.Yellow, st.Cyan, st.Blue = ident, ident, ident, ident, ident
	st.Accent, st.Key = sgr("1"), sgr("1")
	return st
}

// Terminal is the Styler for the terminal: ANSI, or Mono when noColor is set
// (NO_COLOR is present and not empty).
func Terminal(noColor bool) Styler {
	if noColor {
		return Mono()
	}
	return ANSI()
}

// keyed draws a line of markup w cells wide in the base style, with the words
// in braces highlighted as keys: "{Tab} offered · {Enter} take". The braces are
// not drawn; the layout is of the plain text, so the line is cut and padded as
// fit would.
func keyed(markup string, w int, base func(string) string, st Styler) string {
	if w <= 0 {
		return ""
	}
	var segs []struct {
		text string
		key  bool
	}
	var plain strings.Builder
	for rest := markup; rest != ""; {
		open := strings.IndexByte(rest, '{')
		closeAt := strings.IndexByte(rest, '}')
		if open < 0 || closeAt < open {
			segs = append(segs, struct {
				text string
				key  bool
			}{rest, false})
			plain.WriteString(rest)
			break
		}
		if open > 0 {
			segs = append(segs, struct {
				text string
				key  bool
			}{rest[:open], false})
			plain.WriteString(rest[:open])
		}
		segs = append(segs, struct {
			text string
			key  bool
		}{rest[open+1 : closeAt], true})
		plain.WriteString(rest[open+1 : closeAt])
		rest = rest[closeAt+1:]
	}
	cells := []rune(fit(plain.String(), w))
	var b strings.Builder
	for _, s := range segs {
		n := min(len([]rune(s.text)), len(cells))
		if n == 0 {
			continue
		}
		part := string(cells[:n])
		cells = cells[n:]
		if s.key {
			b.WriteString(st.Key(part))
		} else {
			b.WriteString(base(part))
		}
	}
	if len(cells) > 0 {
		b.WriteString(base(string(cells)))
	}
	return b.String()
}

// help is a line of key help: dim, with its keys highlighted.
func help(markup string, w int, st Styler) string { return keyed(markup, w, st.Dim, st) }
