package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The device login (D119, RFC 8628 with JSON bodies): the terminal asks the
// server for a code, the player approves it in the browser at /activate, and the
// terminal's poll then gets a session token. The server's endpoints are under
// /auth/device/ on the same host as the gateway.

// Login errors the caller words for the player.
var (
	ErrLoginDenied  = errors.New("the sign-in was denied in the browser")
	ErrLoginExpired = errors.New("the sign-in code ran out before it was approved")
)

// Device is the server's answer to a start: what to show the player and what
// to poll with.
type Device struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"` // seconds
	Interval                int    `json:"interval"`   // seconds between polls
}

// Grant is an approved login.
type Grant struct {
	Token   string `json:"token"`
	Account string `json:"account"`
}

// DeviceLogin runs the device flow against one server.
type DeviceLogin struct {
	Base string       // the server's origin, https://play.spielengine.com
	HTTP *http.Client // nil: a client with a short timeout
	// Wait sleeps between polls (nil: a real sleep that the context cuts short).
	Wait func(ctx context.Context, d time.Duration) error
	// Shown is called once with the device, after the start, so the caller can
	// print the code and open the browser.
	Shown func(Device)
	// Pending is called on each poll that is not yet answered (a spinner).
	Pending func()
}

// HTTPBase is the http(s) origin of a gateway URL (wss://host:8443/seq/v1 →
// https://host:8443), where the sign-in endpoints are. A gateway on a path of its
// own keeps nothing of it: /auth/ is at the root.
func HTTPBase(gateway string) (string, error) {
	u, err := url.Parse(gateway)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%q is not a server address", gateway)
	}
	switch u.Scheme {
	case "wss", "https":
		return "https://" + u.Host, nil
	case "ws", "http":
		return "http://" + u.Host, nil
	}
	return "", fmt.Errorf("%q is not a ws:// or wss:// address", gateway)
}

// HTTPClient is an HTTP client for the sign-in endpoints, with the same TLS
// settings as the gateway's (-insecure on a local server).
func HTTPClient(cfg *tls.Config) *http.Client {
	return &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: cfg, Proxy: http.ProxyFromEnvironment}}
}

func (d *DeviceLogin) client() *http.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (d *DeviceLogin) wait(ctx context.Context, dur time.Duration) error {
	if d.Wait != nil {
		return d.Wait(ctx, dur)
	}
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// apiError is a refusal of an /auth/ endpoint: {error, message}.
type apiError struct {
	Status  int
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("the server answered %d %s", e.Status, e.Code)
}

func (d *DeviceLogin) post(ctx context.Context, path string, body, out any) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.Base, "/")+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client().Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach the server for sign-in: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusOK {
		return json.Unmarshal(raw, out)
	}
	e := &apiError{Status: resp.StatusCode}
	if json.Unmarshal(raw, e) != nil || (e.Code == "" && e.Message == "") {
		if resp.StatusCode == http.StatusNotFound {
			return errors.New("this server does not offer browser sign-in (set SEQ_SESSION_TOKEN to a session token)")
		}
	}
	return e
}

// Start asks the server for a device code.
func (d *DeviceLogin) Start(ctx context.Context) (Device, error) {
	var dev Device
	if err := d.post(ctx, "/auth/device/start", struct{}{}, &dev); err != nil {
		return dev, err
	}
	if dev.DeviceCode == "" || dev.UserCode == "" {
		return dev, errors.New("the server's sign-in answer had no code")
	}
	if dev.Interval <= 0 {
		dev.Interval = 5
	}
	return dev, nil
}

// Run starts the flow and polls until the player approves (a Grant) or it fails:
// denied (ErrLoginDenied), ran out (ErrLoginExpired), the context ended, or the
// server cannot be reached. It polls at the server's interval and slows down
// by what slow_down asks.
func (d *DeviceLogin) Run(ctx context.Context) (Grant, error) {
	dev, err := d.Start(ctx)
	if err != nil {
		return Grant{}, err
	}
	if d.Shown != nil {
		d.Shown(dev)
	}
	interval := time.Duration(dev.Interval) * time.Second
	for {
		if err := d.wait(ctx, interval); err != nil {
			return Grant{}, err
		}
		var g struct {
			Grant
			AccessToken string `json:"access_token"`
		}
		err := d.post(ctx, "/auth/device/token", map[string]string{"device_code": dev.DeviceCode}, &g)
		if err == nil {
			if g.Token == "" {
				g.Token = g.AccessToken
			}
			if g.Token == "" {
				return Grant{}, errors.New("the server approved the sign-in but sent no session")
			}
			return g.Grant, nil
		}
		var ae *apiError
		if !errors.As(err, &ae) {
			return Grant{}, err
		}
		switch ae.Code {
		case "authorization_pending":
			if d.Pending != nil {
				d.Pending()
			}
		case "slow_down":
			interval += 5 * time.Second
			if d.Pending != nil {
				d.Pending()
			}
		case "expired_token":
			return Grant{}, ErrLoginExpired
		case "access_denied":
			return Grant{}, ErrLoginDenied
		default:
			return Grant{}, ae
		}
	}
}

// IsCredentialRefused says whether err is the server turning down the session
// token at hello (expired, revoked, never valid), as opposed to a connection that
// failed.
func IsCredentialRefused(err error) bool {
	return err != nil && strings.Contains(err.Error(), "hello refused")
}
