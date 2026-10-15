package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/glynternet/packing/internal/hosted"
	"github.com/spf13/cobra"
)

//go:embed hosted.js
var hostedJS []byte

const hostedPage = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Private packing trip</title></head><body><p id="access-status">Opening private trip…</p><div id="app"></div><script src="/elm.js"></script><script src="/hosted.js"></script></body></html>`

func addHostedCommands(root *cobra.Command) {
	var dir string
	command := &cobra.Command{Use: "hosted", Short: "Operate one private, durable packing trip (HTTP MVP)"}
	command.PersistentFlags().StringVar(&dir, "store", "", "private persistent store directory (required)")
	_ = command.MarkPersistentFlagRequired("store")
	withStore := func(fn func(*hosted.Store) error) error {
		s, err := hosted.Open(dir)
		if err != nil {
			return err
		}
		defer s.Close()
		return fn(s)
	}
	var groups, selection string
	init := &cobra.Command{Use: "init", Args: cobra.NoArgs, Short: "Copy pinned definitions and seed an unapplied trip draft once", RunE: func(cmd *cobra.Command, _ []string) error {
		defs, err := hosted.ReadDefinitions(groups)
		if err != nil {
			return err
		}
		text := ""
		if selection != "" {
			f, err := os.Open(selection)
			if err != nil {
				return err
			}
			defer f.Close()
			b, err := io.ReadAll(io.LimitReader(f, hosted.MaxText+1))
			if err != nil {
				return err
			}
			text = string(b)
		}
		return withStore(func(s *hosted.Store) error {
			if err := s.Seed(defs, text); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Trip initialized; issue an access link with hosted access.")
			return nil
		})
	}}
	init.Flags().StringVar(&groups, "groups-dir", "", "flat directory of definition files to pin")
	_ = init.MarkFlagRequired("groups-dir")
	init.Flags().StringVar(&selection, "selection-file", "", "optional draft to seed; originals remain unchanged")
	var accessOrigin, accessOutput string
	access := &cobra.Command{Use: "access", Args: cobra.NoArgs, Short: "Rotate trip access and revoke all browser sessions; stop server first", RunE: func(cmd *cobra.Command, _ []string) error {
		if _, err := publicOrigin(accessOrigin, true); err != nil {
			return err
		}
		// Reserve the output before rotating so a mistyped/existing file cannot
		// revoke access and then fail to deliver its replacement.
		output, err := os.OpenFile(accessOutput, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		defer output.Close()
		err = withStore(func(s *hosted.Store) error {
			secret, err := s.RotateAccess()
			if err != nil {
				return err
			}
			if _, err = fmt.Fprintf(output, "%s/sessions/%s#access=%s\n", accessOrigin, s.Read().ID, secret); err != nil {
				return err
			}
			return output.Sync()
		})
		if err != nil {
			return errors.New("access setup failed; inspect private store/output permissions and issue a fresh link")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Access link written to the requested private file. Anyone holding it can edit and pack this trip. Rotation revoked earlier links and browser sessions.")
		return nil
	}}
	access.Flags().StringVar(&accessOrigin, "origin", "", "public HTTPS origin (or loopback HTTP for testing)")
	access.Flags().StringVar(&accessOutput, "output", "", "new private file for the secret access link")
	_ = access.MarkFlagRequired("origin")
	_ = access.MarkFlagRequired("output")
	var backupFile, restoreFile string
	backup := &cobra.Command{Use: "backup", Args: cobra.NoArgs, Short: "Write a consistent private backup; stop server first", RunE: func(*cobra.Command, []string) error {
		return withStore(func(s *hosted.Store) error { return s.Backup(backupFile) })
	}}
	backup.Flags().StringVar(&backupFile, "output", "", "new backup file")
	_ = backup.MarkFlagRequired("output")
	restore := &cobra.Command{Use: "restore", Args: cobra.NoArgs, Short: "Restore into an empty store; rotate dataset epoch and revoke access", RunE: func(*cobra.Command, []string) error {
		return withStore(func(s *hosted.Store) error { return s.Restore(restoreFile) })
	}}
	restore.Flags().StringVar(&restoreFile, "input", "", "backup file")
	_ = restore.MarkFlagRequired("input")
	var origin, listen string
	var insecure bool
	serve := &cobra.Command{Use: "serve", Args: cobra.NoArgs, Short: "Serve behind a same-host HTTPS proxy; SSE remains milestone 2", RunE: func(*cobra.Command, []string) error {
		if _, err := publicOrigin(origin, insecure); err != nil {
			return err
		}
		host, _, err := net.SplitHostPort(listen)
		if err != nil {
			return err
		}
		if insecure && (net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback()) {
			return errors.New("insecure testing requires a numeric loopback listen address")
		}
		return withStore(func(s *hosted.Store) error {
			if !s.Ready() || s.AccessScope() == "" {
				return errors.New("initialize the store and issue its access link first")
			}
			server := &http.Server{Addr: listen, Handler: hostedHandler(s, origin, insecure), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
			return server.ListenAndServe()
		})
	}}
	serve.Flags().StringVar(&origin, "origin", "", "exact public HTTPS origin; proxy must preserve Origin")
	_ = serve.MarkFlagRequired("origin")
	serve.Flags().StringVar(&listen, "listen", "127.0.0.1:3865", "listen address; keep behind a same-host TLS proxy")
	serve.Flags().BoolVar(&insecure, "insecure-localhost", false, "allow HTTP cookies only on numeric loopback for synthetic tests")
	command.AddCommand(init, access, backup, restore, serve)
	root.AddCommand(command)
}
func publicOrigin(raw string, insecure bool) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("origin must be a bare HTTPS origin")
	}
	if u.Scheme == "https" {
		return u, nil
	}
	ip := net.ParseIP(u.Hostname())
	if insecure && u.Scheme == "http" && ip != nil && ip.IsLoopback() {
		return u, nil
	}
	return nil, errors.New("HTTPS required; HTTP is only supported for explicit loopback testing")
}
func hostedHandler(s *hosted.Store, origin string, insecure bool) http.Handler {
	cookieName := "__Host-packing"
	if insecure {
		cookieName = "packing-local"
	}
	authorized := func(r *http.Request) bool {
		cookie, err := r.Cookie(cookieName)
		return err == nil && s.Authorized(cookie.Value)
	}
	api := hosted.API(s, authorized, origin)
	var mu sync.Mutex
	window := time.Now()
	exchanges := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		if !insecure {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		switch r.URL.Path {
		case "/health/live":
			w.WriteHeader(204)
			return
		case "/health/ready":
			if !s.Ready() {
				w.WriteHeader(503)
			} else {
				w.WriteHeader(204)
			}
			return
		case "/api/v1/exchange":
			w.Header().Set("Content-Type", "application/json")
			if r.Method != "POST" || r.Header.Get("Origin") != origin || r.Header.Get("X-Packing-Request") != "1" {
				http.Error(w, `{"code":"origin_required"}`, 403)
				return
			}
			mu.Lock()
			if time.Since(window) >= time.Minute {
				window = time.Now()
				exchanges = 0
			}
			exchanges++
			limited := exchanges > 20
			mu.Unlock()
			if limited {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"code":"rate_limited"}`, 429)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1024)
			var body struct {
				Secret string `json:"secret"`
			}
			d := json.NewDecoder(r.Body)
			d.DisallowUnknownFields()
			if err := d.Decode(&body); err != nil {
				http.Error(w, `{"code":"invalid_request"}`, 400)
				return
			}
			var extra any
			if d.Decode(&extra) != io.EOF {
				http.Error(w, `{"code":"invalid_request"}`, 400)
				return
			}
			token, err := s.Exchange(body.Secret)
			if err != nil {
				var f *hosted.Fault
				if errors.As(err, &f) {
					http.Error(w, `{"code":"access_required"}`, f.Status)
				} else {
					http.Error(w, `{"code":"unavailable"}`, 503)
				}
				return
			}
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", Secure: !insecure, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(hosted.SessionLifetime.Seconds())})
			w.WriteHeader(204)
			return
		case "/api/v1/access":
			w.Header().Set("Content-Type", "application/json")
			if r.Method != "GET" {
				w.WriteHeader(405)
				return
			}
			if !authorized(r) {
				http.Error(w, `{"code":"access_required"}`, 401)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"trip": s.Read().ID, "scope": s.AccessScope()})
			return
		case "/elm.js", "/hosted.js":
			if r.Method != "GET" {
				w.WriteHeader(405)
				return
			}
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			if r.URL.Path == "/elm.js" {
				_, _ = w.Write(elmJS)
			} else {
				_, _ = w.Write(hostedJS)
			}
			return
		}
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/sessions/") && len(strings.Split(r.URL.Path, "/")) == 3 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, hostedPage)
			return
		}
		api.ServeHTTP(w, r)
	})
}
