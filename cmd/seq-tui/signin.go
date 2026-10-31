package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/GeneStevens/seq-tui/client"
)

// signIn is what `seq-tui login`, `logout` and a plain start share: where the
// saved session lives, how to reach the server's sign-in endpoints, and how to
// show the player the code. The fields are the seams the tests use.
type signIn struct {
	Gateway string       // wss://host/seq/v1
	Dir     string       // the sessions directory (~/.config/seq)
	Out     io.Writer    // what the player reads
	HTTP    *http.Client // for /auth/ (nil: the default)
	Open    func(url string) error
	Now     func() time.Time
	Wait    func(ctx context.Context, d time.Duration) error // nil: real sleeps
}

func (s signIn) path() (string, error) { return client.SessionPath(s.Dir, s.Gateway) }

func (s signIn) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Saved is the saved session for the server, if there is a usable one.
func (s signIn) Saved() (client.Saved, bool) {
	p, err := s.path()
	if err != nil {
		return client.Saved{}, false
	}
	return client.LoadSession(p, s.now())
}

// Login runs the device flow: prints the code and address, tries to open the
// browser, waits for the approval and saves the session.
func (s signIn) Login(ctx context.Context) (client.Saved, error) {
	base, err := client.HTTPBase(s.Gateway)
	if err != nil {
		return client.Saved{}, err
	}
	path, err := s.path()
	if err != nil {
		return client.Saved{}, err
	}
	spin := 0
	d := client.DeviceLogin{Base: base, HTTP: s.HTTP, Wait: s.Wait}
	d.Shown = func(dev client.Device) {
		fmt.Fprintf(s.Out, "To sign in, open\n\n    %s\n\nand approve this code:  %s\n\n", dev.VerificationURI, dev.UserCode)
		if s.Open != nil {
			if err := s.Open(dev.VerificationURIComplete); err != nil {
				fmt.Fprintf(s.Out, "(could not open a browser: open the address above yourself)\n")
			} else {
				fmt.Fprintf(s.Out, "(opened it in your browser)\n")
			}
		}
		fmt.Fprintf(s.Out, "Waiting for you to approve it (Ctrl+C to give up) ")
	}
	d.Pending = func() {
		fmt.Fprintf(s.Out, "\b%c", `|/-\`[spin%4])
		spin++
	}
	g, err := d.Run(ctx)
	fmt.Fprintln(s.Out)
	if err != nil {
		return client.Saved{}, loginWords(err)
	}
	saved := client.Saved{Token: g.Token, Account: g.Account, Saved: s.now().UTC()}
	if err := client.SaveSession(path, saved); err != nil {
		return client.Saved{}, fmt.Errorf("signed in, but the session could not be saved (%v)", err)
	}
	who := ""
	if g.Account != "" {
		who = " as " + g.Account
	}
	fmt.Fprintf(s.Out, "Signed in%s. Saved to %s\n", who, path)
	return saved, nil
}

// loginWords puts a failed login in a sentence for the player.
func loginWords(err error) error {
	switch {
	case errors.Is(err, client.ErrLoginDenied):
		return errors.New("sign-in denied in the browser; run seq-tui login to try again")
	case errors.Is(err, client.ErrLoginExpired):
		return errors.New("the sign-in code expired before it was approved; run seq-tui login to get a new one")
	case errors.Is(err, context.Canceled):
		return errors.New("sign-in cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return errors.New("sign-in timed out")
	}
	return err
}

// Logout forgets the saved session.
func (s signIn) Logout() error {
	p, err := s.path()
	if err != nil {
		return err
	}
	if _, ok := s.Saved(); !ok {
		if _, serr := os.Stat(p); serr != nil {
			fmt.Fprintln(s.Out, "No saved sign-in for this server.")
			return nil
		}
	}
	if err := client.RemoveSession(p); err != nil {
		return err
	}
	fmt.Fprintf(s.Out, "Signed out: removed %s\n", p)
	return nil
}

// Token is the session token for a start: SEQ_SESSION_TOKEN if set (dev and
// tests), else the saved session, else a login. saved says whether the token is
// the saved one (the server may still turn it down, and then Login again).
func (s signIn) Token(ctx context.Context, env string) (token string, saved bool, err error) {
	if env != "" {
		return env, false, nil
	}
	if sv, ok := s.Saved(); ok {
		return sv.Token, true, nil
	}
	sv, err := s.Login(ctx)
	return sv.Token, false, err
}

// openBrowser opens a web page with the system's opener.
func openBrowser(u string) error {
	if pu, err := url.Parse(u); err != nil || (pu.Scheme != "https" && pu.Scheme != "http") {
		return fmt.Errorf("not a web address: %q", u)
	}
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, u)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// tlsFor is the TLS setting of -insecure: a local server's in-memory certificate.
func tlsFor(addr string, insecure bool) (*tls.Config, error) {
	if !insecure {
		return nil, nil
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return &tls.Config{InsecureSkipVerify: true}, nil // a development server's in-memory certificate
	}
	return nil, fmt.Errorf("-insecure is for a local server only, not %s", u.Hostname())
}
