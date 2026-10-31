package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A saved login (D119): the session token the device flow got, in
// ~/.config/seq/<host>.session, so the next run needs no login. The file is
// 0600 in a 0700 directory.

// SavedLifetime is how long the server keeps a session token (90 days); a file
// older than that is not offered, whatever the server would say.
const SavedLifetime = 90 * 24 * time.Hour

// Saved is the content of a session file.
type Saved struct {
	Token   string    `json:"token"`
	Account string    `json:"account,omitempty"`
	Saved   time.Time `json:"saved"`
}

// ConfigDir is where the sessions live: $XDG_CONFIG_HOME/seq, else ~/.config/seq.
func ConfigDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "seq"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".config", "seq"), nil
}

// SessionPath is the file for the server at gateway, named by its host (and port,
// with the colon as an underscore: a file name).
func SessionPath(dir, gateway string) (string, error) {
	u, err := url.Parse(gateway)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%q is not a server address", gateway)
	}
	host := strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(u.Host)
	return filepath.Join(dir, host+".session"), nil
}

// SaveSession writes the file, replacing any earlier one.
func SaveSession(path string, s Saved) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil { // WriteFile's mode is cut by the umask only; an old file keeps its own
		return err
	}
	return os.Rename(tmp, path)
}

// LoadSession reads the file: ok is false when there is none, it cannot be read
// as a session, or it is older than SavedLifetime at now.
func LoadSession(path string, now time.Time) (s Saved, ok bool) {
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &s) != nil || s.Token == "" {
		return Saved{}, false
	}
	if now.Sub(s.Saved) > SavedLifetime {
		return Saved{}, false
	}
	return s, true
}

// RemoveSession deletes the file; no file is not an error.
func RemoveSession(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
