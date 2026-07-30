// Command list_vms exercises the Stratus service layer end-to-end: OIDC login
// via service.LoginWith (which silently authenticates every cloud provider),
// then ListVMs — no TUI, no Wails. It uses the same bootstrap as the app
// (cmd/shared.Init).
//
// Examples:
//
//	go run ./scripts/list_vms            # pick an IdP, list VMs from all providers
//	go run ./scripts/list_vms -idp entra # preselect the "entra" identity
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/muhamm-ad/stratus/cmd/shared"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

func main() {
	idpName := flag.String("idp", "", "identity provider to use (skips the prompt); must exist in config.json")
	timeout := flag.Duration("timeout", 3*time.Minute, "overall timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 1) Same bootstrap as the TUI / desktop app.
	step("bootstrapping service (shared.Init)")
	svc, warnings, err := shared.Init()
	must(err)
	for _, w := range warnings {
		warn("%v", w)
	}

	idpIDs := svc.IdentityProvidersIDs()
	ok("found %d identity provider(s): %s", len(idpIDs), joinIdentityIDs(idpIDs))

	// 2) Choose identity and sign in through the service.
	// LoginWith also auto-connects every registered cloud provider.
	chosen, err := selectIdentityProvider(svc, idpIDs, core.IdentityProviderID(*idpName))
	must(err)
	ok("using identity %q", chosen)

	onCode := func(dc core.DeviceCode) {
		fmt.Println()
		fmt.Println("  ┌────────────────────────────────────────")
		fmt.Printf("  │  open : %s\n", dc.VerificationURI)
		fmt.Printf("  │  code : %s\n", dc.UserCode)
		fmt.Println("  └────────────────────────────────────────")
		fmt.Println()
	}
	if svc.UsesIdentityUsesDeviceFlow(chosen) {
		step("starting device flow — follow the instructions below")
	} else {
		step("opening your browser — complete the sign-in there…")
	}
	idp, err, cpErrors := svc.LoginWith(ctx, chosen, onCode)
	must(err)
	ok("login complete via service.LoginWith")

	// 3) Auth check through the service (not core/auth directly).
	step("checking authentication (service.IsAuthenticated)")
	if !svc.IsAuthenticated() {
		fail("service reports not authenticated after login")
	}
	ok("authenticated as identity %q", svc.GetActiveIdentityProviderID())

	if info, uerr := idp.UserInfo(ctx); uerr != nil {
		warn("UserInfo: %v", uerr)
	} else {
		fmt.Println("\n── user ───────────────────────────────────")
		for _, k := range []string{"name", "preferred_username", "email", "sub"} {
			if v := info[k]; v != "" {
				fmt.Printf("  %-20s %s\n", k, v)
			}
		}
	}

	// 4) Report cloud connect results from LoginWith, then list VMs.
	cloudIDs := svc.GetCloudProvidersIDs()
	if len(cloudIDs) == 0 {
		fail("no cloud providers registered")
	}

	step("cloud connect results (from LoginWith)")
	var connected []core.CloudProviderID
	for _, cp := range cloudIDs {
		if cerr, failed := cpErrors[cp]; failed {
			warn("%s: %v", string(cp), cerr)
			continue
		}
		ok("%s connected", string(cp))
		connected = append(connected, cp)
	}

	var total int
	for _, cp := range connected {
		fmt.Println()
		step("listing VMs on %s (service.ListVMs)", string(cp))
		vms, lerr := svc.ListVMs(ctx, cp)
		if lerr != nil {
			warn("%s: list failed: %v", string(cp), lerr)
			continue
		}
		ok("%s: %d VM(s)", string(cp), len(vms))
		printVMs(vms)
		total += len(vms)
	}

	fmt.Println()
	ok("done ✓ — %d VM(s) across %d connected provider(s)", total, len(connected))
}

// selectIdentityProvider resolves the identity to use: the -idp flag if given, the only one
// if a single provider is configured, otherwise an interactive prompt.
func selectIdentityProvider(svc *service.Service, idpIDs []core.IdentityProviderID, preset core.IdentityProviderID) (core.IdentityProviderID, error) {
	if preset != "" {
		for _, id := range idpIDs {
			if id == preset {
				return id, nil
			}
		}
		return "", fmt.Errorf("identity %q not found (have: %s)", preset, joinIdentityIDs(idpIDs))
	}
	if len(idpIDs) == 1 {
		return idpIDs[0], nil
	}

	fmt.Println("\nSelect an identity provider:")
	for i, id := range idpIDs {
		hint := ""
		if svc.UsesIdentityUsesDeviceFlow(id) {
			hint = "  — device flow"
		}
		fmt.Printf("  [%d] %s%s\n", i+1, id, hint)
	}
	fmt.Print("\nChoice [1]: ")

	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return idpIDs[0], nil
	}
	if idx, err := strconv.Atoi(line); err == nil && idx >= 1 && idx <= len(idpIDs) {
		return idpIDs[idx-1], nil
	}
	for _, id := range idpIDs {
		if strings.EqualFold(string(id), line) {
			return id, nil
		}
	}
	return "", fmt.Errorf("invalid selection %q", line)
}

func joinIdentityIDs(ids []core.IdentityProviderID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = string(id)
	}
	return strings.Join(parts, ", ")
}

func printVMs(vms []core.VM) {
	if len(vms) == 0 {
		fmt.Println("  (none)")
		return
	}
	fmt.Println("\n── vms ────────────────────────────────────")
	fmt.Printf("  %-16s %-24s %-10s %-12s %-16s %s\n",
		"ID", "NAME", "STATE", "TYPE", "REGION", "IP")
	for _, vm := range vms {
		ip := string(vm.PrivateIP)
		if vm.PublicIP != "" {
			ip = string(vm.PublicIP)
		}
		id := vm.ID
		if len(id) > 16 {
			id = id[:13] + "…"
		}
		name := vm.Name
		if len(name) > 24 {
			name = name[:21] + "…"
		}
		fmt.Printf("  %-16s %-24s %-10s %-12s %-16s %s\n",
			id, name, vm.State, vm.Type, vm.Region, ip)
	}
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
