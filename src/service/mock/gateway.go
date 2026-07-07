package mock

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

// Gateway is a fully in-memory implementation of service.Gateway so that
// `go run ./cmd/stratus-tui` works out of the box. Every place a real cloud
// SDK call belongs is marked TODO.
type Gateway struct {
	vms      []service.VM
	sessions []service.Session
	audit    []service.AuditEntry
	seq      int
}

func New() *Gateway { return &Gateway{vms: sampleVMs(), audit: sampleAudit()} }

func (g *Gateway) IdentityProviders() []service.IdP {
	// TODO: read from config.json "identity" section via the identity/oidc package.
	return []service.IdP{
		{Name: "entra", Description: "microsoft entra id — browser + device flow", Usable: true},
		{Name: "okta", Description: "okta sso — device authorization flow", Usable: true},
	}
}

func (g *Gateway) LoginWith(ctx context.Context, name string, onCode func(core.DeviceCode)) (service.Identity, error) {
	// Simulate an RFC 8628 device flow: surface a code, then "complete".
	// TODO: replace with the real identity/oidc PKCE browser flow (RFC 8252)
	// with device-code fallback; call onCode from the flow's callback/channel.
	if onCode != nil {
		onCode(core.DeviceCode{UserCode: "QKZP-DHTW", VerificationURI: "https://id.stratus.dev/activate", Interval: 5 * time.Second})
	}
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return service.Identity{}, ctx.Err()
	}
	return service.Identity{
		User: "abdallah", IdP: name,
		Provider: map[string]string{
			"aws":   "123456789012 · prod",
			"azure": "sub: Stratus-Prod",
			"gcp":   "project: stratus-dev",
		},
	}, nil
}

func (g *Gateway) ActiveIdentity() (service.Identity, bool) {
	return service.Identity{User: "abdallah", IdP: "okta"}, true
}

func (g *Gateway) ProviderUsable(p string) (bool, core.IdentityConstraint) {
	return true, nil
}

func (g *Gateway) ListVMs(ctx context.Context, provider string) ([]service.VM, error) {
	// TODO: fan out to provider plugins THROUGH the service layer.
	if provider == "" {
		return g.vms, nil
	}
	var out []service.VM
	for _, v := range g.vms {
		if v.Provider == provider {
			out = append(out, v)
		}
	}
	return out, nil
}

func (g *Gateway) OpenSession(ctx context.Context, vmID string) (service.SessionSpec, error) {
	var vm service.VM
	for _, v := range g.vms {
		if v.ID == vmID || v.Name == vmID {
			vm = v
		}
	}
	g.seq++
	id := fmt.Sprintf("sess-%d", g.seq)
	spec := service.SessionSpec{SessionID: id, VMName: vm.Name, Provider: vm.Provider}
	// Build the real native-CLI argv. This is where provider knowledge lives.
	switch vm.Provider {
	case "aws": // aws ssm start-session --target i-XXXX --region REGION
		spec.Bin, spec.Args = "aws", []string{"ssm", "start-session", "--target", vm.ID, "--region", vm.Region}
	case "azure": // az network bastion ssh (Bastion) — see report
		spec.Bin, spec.Args = "az", []string{"network", "bastion", "ssh",
			"--name", "stratus-bastion", "--resource-group", "stratus-prod",
			"--target-resource-id", vm.ID, "--auth-type", "AAD"}
	case "gcp": // gcloud compute ssh INSTANCE --tunnel-through-iap --zone --project
		spec.Bin, spec.Args = "gcloud", []string{"compute", "ssh", vm.Name,
			"--tunnel-through-iap", "--zone=" + vm.Region + "-a", "--project=stratus-dev"}
	}
	// For the standalone demo we DON'T want to actually exec a cloud CLI, so the
	// launcher can swap Bin for a harmless echo. TODO: remove in production.
	if _, err := exec.LookPath(spec.Bin); err != nil {
		spec.Bin, spec.Args = "sh", []string{"-c",
			fmt.Sprintf("echo 'stratus: connected to %s via %s (mock shell). type exit to return.'; exec ${SHELL:-sh}", vm.Name, vm.Method)}
	}
	g.sessions = append(g.sessions, service.Session{ID: id, Target: vm.Name, Provider: vm.Provider, Method: vm.Method, Opened: time.Now()})
	return spec, nil
}

func (g *Gateway) StopVM(ctx context.Context, vmID string) error       { return nil } // TODO
func (g *Gateway) CloseSession(ctx context.Context, id string) error {
	for i, s := range g.sessions {
		if s.ID == id {
			g.sessions = append(g.sessions[:i], g.sessions[i+1:]...)
			break
		}
	}
	return nil
}
func (g *Gateway) Sessions() []service.Session       { return g.sessions }
func (g *Gateway) AuditLog() []service.AuditEntry    { return g.audit }

func (g *Gateway) DetectCLIs() []service.CLIStatus {
	// REAL detection with exec.LookPath — this part is production-ready.
	det := func(name, bin, hint string) service.CLIStatus {
		_, err := exec.LookPath(bin)
		return service.CLIStatus{Name: name, Bin: bin, Detected: err == nil, Hint: hint}
	}
	return []service.CLIStatus{
		det("aws-cli", "aws", "install to connect"),
		det("session-manager-plugin", "session-manager-plugin", "install to connect"),
		det("az-cli", "az", "install to connect"),
		det("gcloud", "gcloud", "install to connect"),
	}
}

func sampleVMs() []service.VM {
	tag := func(kv ...string) map[string]string {
		m := map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	}
	act := []service.Activity{{OK: true, When: time.Now().Add(-2 * time.Hour), Text: "dana.k opened SSM session"}}
	return []service.VM{
		{Name: "web-prod-01", ID: "i-0f3a9c12", Provider: "aws", Region: "us-east-1", Type: "t3.large", State: service.StateRunning, PrivateIP: "10.0.1.21", Method: "SSM", Tags: tag("env", "prod", "team", "web", "app", "storefront"), CanConnect: true, Recent: act},
		{Name: "api-prod-02", ID: "i-0a11bb22", Provider: "aws", Region: "us-east-1", Type: "m5.xlarge", State: service.StateRunning, Method: "SSM", CanConnect: true, Tags: tag("env", "prod")},
		{Name: "worker-prod-03", ID: "i-0c33dd44", Provider: "aws", Region: "us-east-1", Type: "c5.2xlarge", State: service.StateStopped, Method: "SSM", CanConnect: true},
		{Name: "db-replica-01", ID: "i-0e55ff66", Provider: "aws", Region: "eu-west-1", Type: "r5.large", State: service.StateRunning, Method: "SSM", CanConnect: false},
		{Name: "bastion-eu", ID: "i-0778899a", Provider: "aws", Region: "eu-west-1", Type: "t3.micro", State: service.StateRunning, Method: "SSM", CanConnect: true},
		{Name: "batch-ap-01", ID: "i-0abbcc01", Provider: "aws", Region: "ap-southeast-2", Type: "c5.large", State: service.StateStarting, Method: "SSM", CanConnect: true},
		{Name: "monitoring-01", ID: "i-0ddeeff2", Provider: "aws", Region: "us-east-1", Type: "t3.medium", State: service.StateRunning, Method: "SSM", CanConnect: true},
		{Name: "web-staging-01", ID: "vm-web-stg-01", Provider: "azure", Region: "eastus", Type: "Standard_D4s_v5", State: service.StateRunning, Method: "Bastion", CanConnect: true},
		{Name: "api-staging-02", ID: "vm-api-stg-02", Provider: "azure", Region: "eastus", Type: "Standard_D2s_v5", State: service.StateRunning, Method: "Bastion", CanConnect: true},
		{Name: "analytics-01", ID: "vm-analytics-01", Provider: "azure", Region: "westeurope", Type: "Standard_E8s_v5", State: service.StateRunning, Method: "Bastion", CanConnect: true},
		{Name: "jumphost-we", ID: "vm-jump-we", Provider: "azure", Region: "westeurope", Type: "Standard_B2s", State: service.StateRunning, Method: "Bastion", CanConnect: true},
		{Name: "ml-train-01", ID: "vm-ml-train-01", Provider: "azure", Region: "westeurope", Type: "Standard_NC6s_v3", State: service.StateStopping, Method: "Bastion", CanConnect: true, Tags: tag("gpu", "v100")},
		{Name: "backup-we", ID: "vm-backup-we", Provider: "azure", Region: "westeurope", Type: "Standard_D2s_v5", State: service.StateStopped, Method: "Bastion", CanConnect: true},
		{Name: "web-dev-01", ID: "gce-web-dev-01", Provider: "gcp", Region: "us-central1", Type: "e2-standard-4", State: service.StateRunning, Method: "IAP", CanConnect: true},
		{Name: "api-dev-02", ID: "gce-api-dev-02", Provider: "gcp", Region: "us-central1", Type: "e2-standard-2", State: service.StateRunning, Method: "IAP", CanConnect: true},
		{Name: "data-pipeline-01", ID: "gce-data-pipe-01", Provider: "gcp", Region: "us-central1", Type: "n2-standard-8", State: service.StateRunning, Method: "IAP", CanConnect: true},
		{Name: "ci-runner-03", ID: "gce-ci-runner-03", Provider: "gcp", Region: "us-central1", Type: "e2-medium", State: service.StateRunning, Method: "IAP", CanConnect: true},
		{Name: "cache-eu-01", ID: "gce-cache-eu-01", Provider: "gcp", Region: "europe-west1", Type: "n2-highmem-2", State: service.StateRunning, Method: "IAP", CanConnect: true},
		{Name: "search-eu-02", ID: "gce-search-eu-02", Provider: "gcp", Region: "europe-west1", Type: "n2-standard-4", State: service.StateRunning, Method: "IAP", CanConnect: false},
		{Name: "gpu-render-01", ID: "gce-gpu-render-01", Provider: "gcp", Region: "europe-west1", Type: "a2-highgpu-1g", State: service.StateUnknown, Method: "IAP", CanConnect: true},
	}
}

func sampleAudit() []service.AuditEntry {
	base := time.Now().Add(-30 * time.Hour)
	mk := func(off time.Duration, u, vm, p, m string, ok bool) service.AuditEntry {
		return service.AuditEntry{When: base.Add(off), User: u, VM: vm, Provider: p, Method: m, Success: ok}
	}
	return []service.AuditEntry{
		mk(0, "dana.k", "web-prod-01", "aws", "SSM", true),
		mk(2*time.Hour, "omar.r", "api-prod-02", "aws", "SSM", true),
		mk(4*time.Hour, "lin.t", "web-staging-01", "azure", "Bastion", true),
		mk(6*time.Hour, "dana.k", "db-replica-01", "aws", "SSM", false),
		mk(9*time.Hour, "omar.r", "ml-train-01", "azure", "Bastion", true),
		mk(12*time.Hour, "lin.t", "web-dev-01", "gcp", "IAP", true),
		mk(15*time.Hour, "dana.k", "cache-eu-01", "gcp", "IAP", true),
		mk(18*time.Hour, "omar.r", "search-eu-02", "gcp", "IAP", false),
		mk(22*time.Hour, "lin.t", "jumphost-we", "azure", "Bastion", true),
		mk(26*time.Hour, "dana.k", "worker-prod-03", "aws", "SSM", true),
	}
}
