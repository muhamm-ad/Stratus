package aws

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openURL opens a URL in the user's default browser. It is the default
// BrowserOpener; tests inject a fake to avoid launching a real browser.
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, *bsd
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("aws: open browser: %w", err)
	}
	return nil
}
