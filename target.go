package main

// backendTarget describes where the client would connect.
// This is a local configuration shell only — no connectivity is attempted.
type backendTarget struct {
	BaseURL string
	Zone    string
	Player  string
	Mode    string
}

// defaultTarget returns local-dev defaults for the backend target.
func defaultTarget() backendTarget {
	return backendTarget{
		BaseURL: "http://localhost:8080",
		Zone:    "qeynos_hills",
		Player:  "p1",
		Mode:    "rt",
	}
}
