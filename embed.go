package stratus

import "embed"

// Frontend is the compiled Wails UI (frontend/dist).
//
//go:embed all:frontend/dist
var Frontend embed.FS
