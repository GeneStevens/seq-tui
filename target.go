package main

import "os"

// envOr returns the value of the environment variable key if non-empty,
// otherwise returns fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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
	DevToken   string // Dev API token (required for dev endpoints)
}

// defaultTarget returns local-dev defaults for the backend target.
// Defaults match the backend's canonical public RT Crushbone instance.
func defaultTarget() backendTarget {
	return backendTarget{
		BaseURL:    envOr("SEQ_BACKEND_URL", "http://localhost:9090"),
		Zone:       envOr("SEQ_ZONE", "crushbone"),
		Mode:       "RT",
		Visibility: "PUBLIC",
		Affinity:   "open",
		Player:     envOr("SEQ_PLAYER", "p1"),
		DevToken:   envOr("SEQ_DEV_TOKEN", "abc"),
	}
}
