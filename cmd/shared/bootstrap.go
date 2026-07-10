package shared

import (
	"fmt"
	"sort"

	"github.com/muhamm-ad/stratus/configs"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/provider/identity/oidc"
	"github.com/muhamm-ad/stratus/internal/service"
)

// Init builds the shared Service: one generic OIDC identity provider per
// configured "identity" entry, plus every registered + configured connector.
// warnings lists sections skipped due to incomplete config (the app still runs
// with the rest); a non-nil error is fatal (bad config.json or zero identities).
func Init() (svc *service.Service, warnings []error, err error) {
	secs, err := configs.Load()
	if err != nil {
		return nil, nil, err
	}

	idps := make(map[core.IdentityProviderID]core.IdentityProvider)
	ids := make([]core.IdentityProviderID, 0, len(secs.Identity))
	for id := range secs.Identity {
		ids = append(ids, core.IdentityProviderID(id))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		cfg, perr := oidc.ParseConfig(secs.Identity[string(id)])
		if perr != nil {
			warnings = append(warnings, fmt.Errorf("identity %q: %w", id, perr))
			continue
		}
		idps[id] = oidc.New(id, cfg)
	}

	reg, provWarn := core.BuildAll(core.Factories(), secs.Providers)
	warnings = append(warnings, provWarn...)

	if len(idps) == 0 {
		return nil, warnings, fmt.Errorf(`service: no identity provider configured — fill "identity" in config.json`)
	}
	return service.NewService(idps, reg), warnings, nil
}