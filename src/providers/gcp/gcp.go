// Package gcp implements core.ProviderConnector for GCP: it exchanges
// the Entra id_token for a short-lived GCP access token via the Security Token
// Service (OAuth 2.0 Token Exchange, RFC 8693), then uses that token to list
// Compute Engine instances via the REST API.
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
)

const (
	defaultSTSEndpoint     = "https://sts.googleapis.com/v1/token"
	defaultComputeEndpoint = "https://compute.googleapis.com/compute/v1"
)

// Provider implements core.ProviderConnector for GCP.
type Provider struct {
	cfg             Config
	httpClient      *http.Client
	stsEndpoint     string
	computeEndpoint string
	now             func() time.Time

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

type Option func(*Provider)

func WithHTTPClient(c *http.Client) Option      { return func(p *Provider) { p.httpClient = c } }
func WithSTSEndpoint(url string) Option          { return func(p *Provider) { p.stsEndpoint = url } }
func WithComputeEndpoint(url string) Option      { return func(p *Provider) { p.computeEndpoint = url } }
func WithClock(f func() time.Time) Option        { return func(p *Provider) { p.now = f } }

func New(cfg Config, opts ...Option) *Provider {
	p := &Provider{
		cfg:             cfg,
		httpClient:      http.DefaultClient,
		stsEndpoint:     defaultSTSEndpoint,
		computeEndpoint: defaultComputeEndpoint,
		now:             time.Now,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) ID() core.ProviderID { return core.ProviderGCP }

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token != "" && p.now().Before(p.expiresAt)
}

// Authenticate exchanges the Entra id_token for a GCP access token.
func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	idToken, err := idp.IDToken(ctx)
	if err != nil {
		return fmt.Errorf("gcp: %w", err)
	}
	form := url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"audience":             {p.cfg.WorkforceAudience},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:jwt"},
		"subject_token":        {idToken},
		"scope":                {p.cfg.Scope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.stsEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: gcp sts request: %v", core.ErrExchange, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: gcp sts returned %d: %s", core.ErrExchange, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("%w: decode gcp sts response: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.token = out.AccessToken
	p.expiresAt = p.now().Add(time.Duration(out.ExpiresIn) * time.Second)
	p.mu.Unlock()
	return nil
}

// AccessToken returns the held GCP access token.
func (p *Provider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.token = ""
	p.expiresAt = time.Time{}
	p.mu.Unlock()
	return nil
}

// --- Phase 3: ListInstances via Compute Engine REST aggregatedInstances ---

// gcpAggregatedList is the response shape for instances.aggregatedList.
type gcpAggregatedList struct {
	Items         map[string]gcpZoneItems `json:"items"`
	NextPageToken string                  `json:"nextPageToken"`
}

type gcpZoneItems struct {
	Instances []gcpInstance `json:"instances"`
}

type gcpInstance struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	MachineType       string            `json:"machineType"`
	Status            string            `json:"status"`
	Zone              string            `json:"zone"`
	NetworkInterfaces []gcpNetworkIface `json:"networkInterfaces"`
	Labels            map[string]string `json:"labels"`
	CreationTimestamp string            `json:"creationTimestamp"`
}

type gcpNetworkIface struct {
	NetworkIP     string            `json:"networkIP"`
	AccessConfigs []gcpAccessConfig `json:"accessConfigs"`
}

type gcpAccessConfig struct {
	NatIP string `json:"natIP"`
}

// ListInstances lists all Compute Engine instances across zones for the project.
func (p *Provider) ListInstances(ctx context.Context, _ string) ([]core.Instance, error) {
	p.mu.Lock()
	token := p.token
	p.mu.Unlock()
	if token == "" {
		return nil, core.ErrNotAuthenticated
	}

	baseURL := p.computeEndpoint + "/projects/" + p.cfg.ProjectID + "/aggregatedInstances"
	var instances []core.Instance
	pageToken := ""

	for {
		reqURL := baseURL
		if pageToken != "" {
			reqURL += "?pageToken=" + url.QueryEscape(pageToken)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("gcp: build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("gcp: list instances request: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("gcp: list instances returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var page gcpAggregatedList
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("gcp: decode list instances response: %w", err)
		}
		for _, zoneItems := range page.Items {
			for _, gi := range zoneItems.Instances {
				instances = append(instances, mapGCPInstance(gi))
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return instances, nil
}

func mapGCPInstance(gi gcpInstance) core.Instance {
	// Zone is e.g. "zones/us-central1-a" → strip prefix → "us-central1-a"
	zone := path.Base(gi.Zone)
	// Region: drop the trailing "-X" from zone name → "us-central1"
	region := zone
	if idx := strings.LastIndex(zone, "-"); idx > 0 {
		region = zone[:idx]
	}

	// MachineType is e.g. "zones/us-central1-a/machineTypes/n2-standard-4"
	machineType := path.Base(gi.MachineType)

	// State mapping
	state := mapGCPState(gi.Status)

	// Network
	var privateIP, publicIP string
	if len(gi.NetworkInterfaces) > 0 {
		privateIP = gi.NetworkInterfaces[0].NetworkIP
		if len(gi.NetworkInterfaces[0].AccessConfigs) > 0 {
			publicIP = gi.NetworkInterfaces[0].AccessConfigs[0].NatIP
		}
	}

	// Tags come from labels on GCP
	tags := gi.Labels

	return core.Instance{
		ID:           gi.ID,
		Name:         gi.Name,
		State:        state,
		Platform:     "linux", // GCP Compute defaults to Linux; Windows requires checking image
		InstanceType: machineType,
		PrivateIP:    privateIP,
		PublicIP:     publicIP,
		Region:       region,
		OSUser:       "ubuntu", // Most GCP images use ubuntu; override via labels["stratus-os-user"]
		Tags:         tags,
	}
}

func mapGCPState(status string) string {
	switch strings.ToUpper(status) {
	case "RUNNING":
		return "running"
	case "STOPPED", "TERMINATED", "SUSPENDED":
		return "stopped"
	case "PROVISIONING", "STAGING", "SUSPENDING", "REPAIRING":
		return "transitioning"
	default:
		return "unknown"
	}
}

func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

var _ core.ProviderConnector = (*Provider)(nil)
