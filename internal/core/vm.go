package core

import "time"

type VMState string

const (
	StateRunning  VMState = "running"
	StateStopped  VMState = "stopped"
	StateStarting VMState = "starting"
	StateStopping VMState = "stopping"
	StateUnknown  VMState = "unknown"
)

type VMPlatform string

const (
	PlatformLinux   VMPlatform = "linux"
	PlatformWindows VMPlatform = "windows"
)

type VMType string
type VMRegion string
type VMOsUser string
type IPAddress string

// VM is a connectable virtual machine, normalized across providers.
type VM struct {
	ID         string
	Name       string
	State      VMState
	Provider   CloudProviderID
	Platform   VMPlatform
	Type       VMType // machine size, e.g. t3.large / Standard_D4s_v5 / e2-standard-4
	PrivateIP  IPAddress
	PublicIP   IPAddress
	Region     VMRegion
	OSUser     VMOsUser // default OS login user, when known
	LaunchTime time.Time
	Tags       map[string]string
}

// IsRunning reports whether the instance can currently accept connections.
func (i VM) IsRunning() bool { return i.State == "running" }

// IsWindows reports whether the instance runs Windows (RDP) vs Linux (SSH).
func (i VM) IsWindows() bool { return i.Platform == "windows" }
