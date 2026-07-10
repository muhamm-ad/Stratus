// Command authcheck exercises the Stratus OIDC login end-to-end against a real
// identity provider — no TUI, no Wails. It reads the SAME config.json the app
// uses (config.Load), lets you pick one of the configured identity providers,
// and runs the real core/auth flow. A green run means the auth layer works.
//
// Examples:
//
//	go run ./cmd/authcheck                 # pick an IdP interactively (browser flow)
//	go run ./cmd/authcheck -idp entra      # preselect the "entra" identity
//	go run ./cmd/authcheck -device         # force the RFC 8628 device flow
//	go run ./cmd/authcheck -arm            # also test the Azure ARM re-scope
//
// Register the client as a PUBLIC/native app (no secret) and allow a loopback
// redirect (http://127.0.0.1, any port) for the browser flow.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/muhamm-ad/stratus/configs"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/core/auth"
	"github.com/muhamm-ad/stratus/internal/provider/identity/oidc"
)

func main() {
	idpName := flag.String("idp", "", "identity provider to use (skips the prompt); must exist in config.json")
	deviceFlag := flag.Bool("device", false, "force the RFC 8628 device flow instead of the browser")
	arm := flag.Bool("arm", false, "after login, re-scope the refresh token to an ARM access token")
	armScope := flag.String("arm-scope", "https://management.azure.com/.default", "scope for the ARM re-scope test")
	timeout := flag.Duration("timeout", 3*time.Minute, "overall timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 1) Load the same config the app uses.
	step("loading config (config.Load)")
	secs, err := configs.Load()
	must(err)
	names := make([]string, 0, len(secs.Identity))
	for name := range secs.Identity {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		fail(`no identity provider configured — fill the "identity" section of config.json`)
	}
	ok("found %d identity provider(s): %s", len(names), strings.Join(names, ", "))

	// 2) Choose one.
	// chosen, err := chooseIdP(names, *idpName, secs)
	chosen, err := chooseIdP(names, *idpName, secs)
	must(err)

	cfg, err := oidc.ParseConfig(secs.Identity[chosen])
	must(err)
	ok("using identity %q (client_id %s)", chosen, cfg.ClientID)

	// 3) Build the auth client (discovery when an issuer is set, else endpoints).
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "offline_access", "profile", "email"}
	}
	var client *auth.Client
	if cfg.Issuer != "" {
		step("discovering %s", cfg.Issuer)
		client, err = auth.NewClient(ctx, cfg.Issuer, cfg.ClientID, scopes)
		must(err)
	} else {
		client = auth.NewClientManual(cfg.ClientID, cfg.AuthorizeEndpoint, cfg.TokenEndpoint, scopes)
		ok("using explicit endpoints (no discovery)")
	}
	ok("auth endpoint : %s", client.OAuth.Endpoint.AuthURL)
	ok("token endpoint: %s", client.OAuth.Endpoint.TokenURL)
	if client.OAuth.Endpoint.DeviceAuthURL != "" {
		ok("device endpoint: %s", client.OAuth.Endpoint.DeviceAuthURL)
	}

	// 4) Login (browser+loopback, or device flow).
	device := *deviceFlag || cfg.UseDeviceFlow
	var tok *oauth2.Token
	if device {
		if client.OAuth.Endpoint.DeviceAuthURL == "" {
			fail("this issuer does not advertise a device_authorization_endpoint")
		}
		step("starting device flow — follow the instructions below")
		tok, err = client.DeviceLogin(ctx, func(dc core.DeviceCode) {
			fmt.Println()
			fmt.Println("  ┌────────────────────────────────────────")
			fmt.Printf("  │  open : %s\n", dc.VerificationURI)
			fmt.Printf("  │  code : %s\n", dc.UserCode)
			fmt.Println("  └────────────────────────────────────────")
			fmt.Println()
		})
	} else {
		step("opening your browser — complete the sign-in there…")
		tok, err = client.BrowserLogin(ctx)
	}
	must(err)
	ok("login complete")

	// 5) Token summary.
	rawID, _ := tok.Extra("id_token").(string)
	fmt.Println("\n── token ──────────────────────────────────")
	fmt.Printf("  access_token  : %s (%d chars)\n", present(tok.AccessToken), len(tok.AccessToken))
	fmt.Printf("  refresh_token : %s\n", present(tok.RefreshToken))
	fmt.Printf("  token_type    : %s\n", tok.TokenType)
	if !tok.Expiry.IsZero() {
		fmt.Printf("  expires_in    : %s\n", time.Until(tok.Expiry).Round(time.Second))
	}
	fmt.Printf("  id_token      : %s (%d chars)\n", present(rawID), len(rawID))

	// 6) Verify the id_token signature (JWKS) and show a few claims.
	if rawID != "" && client.Verifier != nil {
		step("verifying id_token signature via JWKS")
		idt, verr := client.Verifier.Verify(ctx, rawID)
		must(verr)
		var claims map[string]any
		_ = idt.Claims(&claims)
		ok("id_token verified")
		fmt.Println("\n── id_token claims ────────────────────────")
		for _, k := range []string{"iss", "aud", "sub", "name", "preferred_username", "email", "tid"} {
			if v, has := claims[k]; has {
				fmt.Printf("  %-20s %v\n", k, v)
			}
		}
	}

	// 7) Optional: the one hand-written grant — re-scope to ARM (Azure leg).
	if *arm {
		fmt.Println()
		if tok.RefreshToken == "" {
			warn("no refresh_token — request scope 'offline_access' to test ARM re-scope")
		} else {
			step("re-scoping refresh token to %s (auth.RefreshGrant)", *armScope)
			armTok, aerr := auth.RefreshGrant(ctx, http.DefaultClient,
				client.OAuth.Endpoint.TokenURL, cfg.ClientID, tok.RefreshToken, []string{*armScope})
			if aerr != nil {
				warn("ARM re-scope failed: %v", aerr)
			} else {
				ok("ARM token acquired: %s (%d chars), expires_in %s",
					present(armTok.AccessToken), len(armTok.AccessToken), time.Until(armTok.Expiry).Round(time.Second))
			}
		}
	}

	fmt.Println()
	ok("done ✓")
}

// chooseIdP resolves the identity to use: the -idp flag if given, the only one
// if a single provider is configured, otherwise an interactive prompt.
func chooseIdP(names []string, preset string, secs configs.Sections) (string, error) {
	if preset != "" {
		for _, n := range names {
			if n == preset {
				return n, nil
			}
		}
		return "", fmt.Errorf("identity %q not found in config.json (have: %s)", preset, strings.Join(names, ", "))
	}
	if len(names) == 1 {
		return names[0], nil
	}

	fmt.Println("\nSelect an identity provider:")
	for i, n := range names {
		hint := ""
		if cfg, err := oidc.ParseConfig(secs.Identity[n]); err == nil && cfg.Issuer != "" {
			hint = "  — " + cfg.Issuer
		}
		fmt.Printf("  [%d] %s%s\n", i+1, n, hint)
	}
	fmt.Print("\nChoice [1]: ")

	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return names[0], nil
	}
	if idx, err := strconv.Atoi(line); err == nil && idx >= 1 && idx <= len(names) {
		return names[idx-1], nil
	}
	for _, n := range names {
		if strings.EqualFold(n, line) {
			return n, nil
		}
	}
	return "", fmt.Errorf("invalid selection %q", line)
}

// ── tiny console helpers ────────────────────────────────────────────────────

func step(f string, a ...any) { fmt.Printf("\n▸ "+f+"\n", a...) }
func ok(f string, a ...any)   { fmt.Printf("  ✓ "+f+"\n", a...) }
func warn(f string, a ...any) { fmt.Printf("  ⚠ "+f+"\n", a...) }

func fail(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "  ✗ "+f+"\n", a...)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		fail("%v", err)
	}
}

func present(s string) string {
	if s == "" {
		return "—"
	}
	return "present"
}