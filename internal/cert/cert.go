package cert

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const systemCertDir = "/usr/local/share/ca-certificates"

// Install copies the certificate into the system trust store and refreshes it.
// Requires elevated privileges — prompts via pkexec.
func Install(srcPath string) error {
	if srcPath == "" {
		return nil
	}

	name := filepath.Base(srcPath)
	ext := filepath.Ext(name)
	if !strings.EqualFold(ext, ".crt") {
		name = name[:len(name)-len(ext)] + ".crt"
	}
	destPath := filepath.Join(systemCertDir, name)

	script := fmt.Sprintf("cp '%s' '%s' && update-ca-certificates", srcPath, destPath)
	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation certificat: %w\n%s", err, out)
	}
	return nil
}
