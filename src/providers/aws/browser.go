package aws

// import (
// 	"fmt"
// 	"os/exec"
// 	"runtime"
// )

// // openURL opens a URL in the user's default browser. It is the default
// // BrowserOpener; tests inject a fake to avoid launching a real browser.
// func openURL(url string) error {
// 	var cmd *exec.Cmd
// 	switch runtime.GOOS {
// 	case "windows":
// 		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
// 	case "darwin":
// 		cmd = exec.Command("open", url)
// 	default: // linux, *bsd
// 		cmd = exec.Command("xdg-open", url)
// 	}
// 	if err := cmd.Start(); err != nil {
// 		return fmt.Errorf("aws: open browser: %w", err)
// 	}
// 	return nil
// }


import "github.com/pkg/browser"

// OpenURL opens a URL in the user's default browser. It is the default
// BrowserOpener; tests inject a fake via WithBrowserOpener, and the Wails app
// may inject one backed by runtime.BrowserOpenURL.
//
// We delegate to github.com/pkg/browser rather than shelling out by hand
// because it handles the platform quirks that matter for a long OAuth
// authorization URL: on Windows it calls the Win32 ShellExecute API (robust
// with query-heavy URLs), and on Linux it falls back across xdg-open,
// x-www-browser and www-browser instead of assuming xdg-open exists.
func OpenURL(url string) error {
	return browser.OpenURL(url)
}
