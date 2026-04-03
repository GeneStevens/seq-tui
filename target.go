package main

// backendTarget describes the intended backend connection target.
// This is a local configuration shell only — no connectivity is attempted.
// Fields align with the backend's canonical ZoneInstanceKey routing model.
type backendTarget struct {
	BaseURL    string // Backend HTTP base URL
	Zone       string // Zone name (e.g., "crushbone")
	Mode       string // Zone mode: "RT" or "ASYNC"
	Visibility string // Instance visibility: "PUBLIC" or "PRIVATE"
	Affinity   string // Instance affinity key (e.g., "open" for public)
	Player     string // Player identifier for dev integration
}

// defaultTarget returns local-dev defaults for the backend target.
// Defaults match the backend's canonical public RT Crushbone instance.
func defaultTarget() backendTarget {
	return backendTarget{
		BaseURL:    "http://localhost:9090",
		Zone:       "crushbone",
		Mode:       "RT",
		Visibility: "PUBLIC",
		Affinity:   "open",
		Player:     "p1",
	}
}
