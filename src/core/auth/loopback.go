package auth

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// errLoopbackUnavailable signals that a 127.0.0.1 listener could not be bound,
// so the caller should fall back to the device authorization flow.
var errLoopbackUnavailable = errors.New("aws: loopback listener unavailable")

const callbackPath = "/oauth/callback"

// callbackHTML is shown in the browser tab after the redirect completes.
const callbackHTML = `<!doctype html><html><head><meta charset="utf-8">
<title>Stratus</title></head><body style="font-family:system-ui;text-align:center;margin-top:4rem">
<h2>Login complete</h2><p>You can close this tab and return to Stratus.</p></body></html>`

// listenLoopback binds an ephemeral port on 127.0.0.1 (never 0.0.0.0) and
// returns the listener together with the redirect URI the client must register.
func listenLoopback() (net.Listener, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", errLoopbackUnavailable
	}
	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := "http://127.0.0.1:" + itoa(port) + callbackPath
	return ln, redirectURI, nil
}

// waitForCode serves a single callback request on ln, validates the CSRF state,
// and returns the authorization code. It always shuts the server down before
// returning and honours ctx cancellation (the user pressing Cancel).
func waitForCode(ctx context.Context, ln net.Listener, expectedState string) (string, error) {
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if s := q.Get("state"); s != expectedState {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			done <- result{err: errors.New("aws: oauth state mismatch")}
			return
		}
		if e := q.Get("error"); e != "" {
			http.Error(w, e, http.StatusBadRequest)
			done <- result{err: errors.New("aws: authorization error: " + e)}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			done <- result{err: errors.New("aws: missing authorization code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(callbackHTML))
		done <- result{code: code}
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	select {
	case res := <-done:
		return res.code, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// itoa avoids importing strconv just for one positive-int conversion.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
