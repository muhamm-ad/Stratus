// FIXME: To be deleted after the session management is implemented and all the related code is updated

package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)



type VMState string

const (
	StateRunning  VMState = "running"
	StateStopped  VMState = "stopped"
	StateStarting VMState = "starting"
	StateStopping VMState = "stopping"
	StateUnknown  VMState = "unknown"
)

type VM struct {
	Name, ID, Provider, Region, Type string
	State                            VMState
	PrivateIP                        string
	Method                           string // SSM / Bastion / IAP
	Tags                             map[string]string
	CanConnect                       bool
	Recent                           []Activity
}

type Activity struct {
	OK   bool
	When time.Time
	Text string
}


type Gateway struct {
	svc *service.Service
}

func (g *Gateway) ListVMs(ctx context.Context, provider string) ([]VM, error) {
	if provider == "" {
		var all []VM
		for _, id := range g.svc.CloudProviders() {
			vms, err := g.listProvider(ctx, string(id))
			if err != nil {
				return nil, err
			}
			all = append(all, vms...)
		}
		return all, nil
	}
	return g.listProvider(ctx, provider)
}

func (g *Gateway) listProvider(ctx context.Context, provider string) ([]VM, error) {
	pid := core.CloudProviderID(provider)
	if !g.svc.ProviderUsable(pid) {
		return nil, fmt.Errorf("%w: %s incompatible with active identity", core.ErrExchange, provider)
	}
	if err := g.svc.Connect(ctx, pid); err != nil {
		if errors.Is(err, core.ErrNotAuthenticated) || errors.Is(err, core.ErrExchange) {
			return nil, err
		}
		// Config/connection issues: return empty, not fatal.
		return nil, nil
	}
	instances, err := g.svc.ListInstances(ctx, pid, g.svc.Accounts()[pid])
	if err != nil {
		if errors.Is(err, core.ErrNotImplemented) {
			return nil, nil
		}
		if errors.Is(err, core.ErrNotAuthenticated) || errors.Is(err, core.ErrExchange) {
			return nil, err
		}
		return nil, nil
	}
	out := make([]VM, len(instances))
	for i, inst := range instances {
		out[i] = instanceToVM(inst, provider)
	}
	return out, nil
}

func instanceToVM(inst core.Instance, provider string) VM {
	method := "SSM"
	switch provider {
	case "azure":
		method = "Bastion"
	case "gcp":
		method = "IAP"
	}
	return VM{
		Name:       inst.Name,
		ID:         inst.ID,
		Provider:   provider,
		Region:     inst.Region,
		Type:       inst.InstanceType,
		State:      mapInstanceState(inst.State),
		PrivateIP:  inst.PrivateIP,
		Method:     method,
		Tags:       inst.Tags,
		CanConnect: inst.IsRunning(),
	}
}

func mapInstanceState(s string) VMState {
	switch strings.ToLower(s) {
	case "running":
		return StateRunning
	case "stopped", "terminated":
		return StateStopped
	case "pending", "starting":
		return StateStarting
	case "stopping":
		return StateStopping
	default:
		return StateUnknown
	}
}

