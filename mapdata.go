package main

const (
	// Player marker and fixed position for static rendering.
	playerMarker = '@'
	playerX      = 5
	playerY      = 3

	// awarenessRadius is the visual dimming radius around the player.
	// Tiles beyond this distance appear faint. Purely aesthetic.
	awarenessRadius = 8
)

// landmark represents a static ambient map feature.
type landmark struct {
	x     int
	y     int
	glyph rune
	label string
}

// landmarks are hardcoded environmental features for visual character.
// These are not authoritative entities — purely presentation.
var landmarks = []landmark{
	{x: 14, y: 1, glyph: '*', label: "torch"},
	{x: 10, y: 9, glyph: '~', label: "pool"},
	{x: 25, y: 6, glyph: '>', label: "stairs"},
	{x: 3, y: 18, glyph: '+', label: "shrine"},
}

// threatMarker represents a static ambient threat-presence cue.
type threatMarker struct {
	x     int
	y     int
	glyph rune
	label string
}

// threatMarkers are hardcoded atmospheric danger cues.
// These are not authoritative entities — purely presentation.
var threatMarkers = []threatMarker{
	{x: 20, y: 4, glyph: '!', label: "uneasy presence"},
	{x: 7, y: 12, glyph: '?', label: "faint scratching"},
	{x: 16, y: 17, glyph: '!', label: "distant scraping"},
}

// staticMap is a hardcoded ASCII map for the initial rendering slice.
// In the future, map data will come from the backend.
const staticMap = `##############################
#............................#
#..####..........##..........#
#..#  #..........##..........#
#..#  #..........................
#..####..........##..........#
#................##..........#
#............................#
#........########............#
#........#      #............#
#........#      #............#
#........########............#
#............................#
#............................#
#..........####..............#
#..........#  #......##......#
#..........#  #......##......#
#..........####..............#
#............................#
##############################`
