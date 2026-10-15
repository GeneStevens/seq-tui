package screen

// Styler decorates text for a terminal. Layout is always computed on plain
// text first and styled after, so a style never changes a width. Colour is
// only ever an extra: every distinction a style makes is also made by a glyph
// or a word (doc 01, accessibility).
type Styler struct {
	Bold, Dim, Reverse func(string) string
	Red, Green, Yellow func(string) string
	Cyan               func(string) string
}

func ident(s string) string { return s }

// Plain is the Styler that adds nothing: golden screens and replays use it.
func Plain() Styler {
	return Styler{ident, ident, ident, ident, ident, ident, ident}
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
// which the terminal maps to its own palette.
func ANSI() Styler {
	return Styler{Bold: sgr("1"), Dim: sgr("2"), Reverse: sgr("7"), Red: sgr("31"), Green: sgr("32"), Yellow: sgr("33"), Cyan: sgr("36")}
}
