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
	// Magenta is the colour of a world event's marks, a colour no person or quest
	// mark has. Alert is a refusal on the status line: bold white on red, which
	// stays a reverse bold line without colour.
	Magenta, Alert func(string) string
	// The chat palette (D101 §8), in one place so the OOC channel and the colour of
	// items can extend it: one colour for each channel, and for each kind of name.
	// Mention is the line that names the character. Colour is never the only sign:
	// tags, the » of a mention and the underlined own name stay without it.
	Chat ChatPalette
}

// ChatPalette is what chat is drawn with: a style for each channel and for each
// kind of name.
type ChatPalette struct {
	Say, Party, Tell, Server func(string) string // the channels
	Me, Player, NPC, Neutral func(string) string // the names
	Mention                  func(string) string // a line that names the character
}

func ident(s string) string { return s }

// plainChat is the chat palette that adds nothing.
func plainChat() ChatPalette {
	return ChatPalette{Say: ident, Party: ident, Tell: ident, Server: ident, Me: ident, Player: ident, NPC: ident, Neutral: ident, Mention: ident}
}

// Plain is the Styler that adds nothing: golden screens and replays use it.
func Plain() Styler {
	return Styler{Bold: ident, Dim: ident, Reverse: ident, Red: ident, Green: ident, Yellow: ident, Cyan: ident, Blue: ident, Accent: ident, Key: ident, Magenta: ident, Alert: ident, Chat: plainChat()}
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
		Accent: sgr("1;36"), Key: sgr("36"), Magenta: sgr("35"), Alert: sgr("1;97;41"),
		Chat: ChatPalette{
			Say: ident, Party: sgr("36"), Tell: sgr("33"), Server: sgr("1;33"),
			Me: sgr("1;32"), Player: sgr("94"), NPC: sgr("35"), Neutral: ident,
			Mention: sgr("1"),
		}}
}

// Mono is ANSI without colour, for NO_COLOR (no-color.org): the hierarchy is
// kept with bold, dim and reverse, which are not colours.
func Mono() Styler {
	st := ANSI()
	st.Red, st.Green, st.Yellow, st.Cyan, st.Blue, st.Magenta = ident, ident, ident, ident, ident, ident
	st.Alert = sgr("1;7")
	st.Accent, st.Key = sgr("1"), sgr("1")
	// Without colour the channels are told by their tags; the server's lines are
	// bold and the character's own name underlined.
	st.Chat = plainChat()
	st.Chat.Server, st.Chat.Me, st.Chat.Mention = sgr("1"), sgr("1;4"), sgr("1")
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

// warnOpen and warnClose bracket a warning in keyed markup: a refusal or a shut
// door in the middle of a line of hints.
const (
	warnOpen  = "\x01"
	warnClose = "\x02"
)

// keyed draws a line of markup w cells wide in the base style, with the words
// in braces highlighted as keys: "{Tab} offered · {Enter} take", and the words
// between warnOpen and warnClose in red. The marks are not drawn; the layout is
// of the plain text, so the line is cut and padded as fit would.
func keyed(markup string, w int, base func(string) string, st Styler) string {
	if w <= 0 {
		return ""
	}
	type seg struct {
		text string
		fn   func(string) string
	}
	var segs []seg
	var plain strings.Builder
	cur, fn := strings.Builder{}, base
	flush := func() {
		if cur.Len() > 0 {
			segs = append(segs, seg{cur.String(), fn})
			plain.WriteString(cur.String())
			cur.Reset()
		}
	}
	for _, r := range markup {
		switch string(r) {
		case "{":
			flush()
			fn = st.Key
		case "}":
			flush()
			fn = base
		case warnOpen:
			flush()
			fn = st.Red
		case warnClose:
			flush()
			fn = base
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	cells := []rune(fit(plain.String(), w))
	var b strings.Builder
	for _, s := range segs {
		n := min(len([]rune(s.text)), len(cells))
		if n == 0 {
			continue
		}
		b.WriteString(s.fn(string(cells[:n])))
		cells = cells[n:]
	}
	if len(cells) > 0 {
		b.WriteString(base(string(cells)))
	}
	return b.String()
}

// help is a line of key help: dim, with its keys highlighted.
func help(markup string, w int, st Styler) string { return keyed(markup, w, st.Dim, st) }
