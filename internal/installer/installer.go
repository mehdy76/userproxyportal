package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	BinDir  = "/usr/local/bin"
	ConfDir = "/etc/userproxyportal"
)

// UserSystemdDir retourne ~/.config/systemd/user/ pour l'utilisateur courant.
func UserSystemdDir() string {
	if cfg := os.Getenv("XDG_CONFIG_HOME"); cfg != "" {
		return filepath.Join(cfg, "systemd", "user")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

type ComponentStatus struct {
	Name      string
	Installed bool
	Path      string
}

func CheckComponents() []ComponentStatus {
	svcPath := filepath.Join(UserSystemdDir(), "userproxyportal.service")
	items := []struct{ name, path string }{
		{"userproxyportal", BinDir + "/userproxyportal"},
		{"userproxyportal.service", svcPath},
		{"config.yaml", ConfDir + "/config.yaml"},
	}
	statuses := make([]ComponentStatus, len(items))
	for i, it := range items {
		_, err := os.Stat(it.path)
		statuses[i] = ComponentStatus{Name: it.name, Path: it.path, Installed: err == nil}
	}
	return statuses
}

// SelfInstall installe le binaire et la config via pkexec (root),
// puis installe le service dans ~/.config/systemd/user/ sans élévation.
func SelfInstall(exePath string, serviceContent, configContent []byte) error {
	// --- Partie root : binaire + répertoire config ---
	cfgTmp, err := writeTempFile("*.yaml", configContent)
	if err != nil {
		return err
	}
	defer os.Remove(cfgTmp)

	binDest := BinDir + "/userproxyportal"
	cfgDest := ConfDir + "/config.yaml"

	rootScript := fmt.Sprintf(`set -e
cp '%s' '%s' && chmod 755 '%s'
mkdir -p '%s'
[ -f '%s' ] || cp '%s' '%s'
`, exePath, binDest, binDest,
		ConfDir, cfgDest, cfgTmp, cfgDest)

	out, err := exec.Command("pkexec", "sh", "-c", rootScript).CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation (root): %w\n%s", err, out)
	}

	// --- Partie utilisateur : fichier service ---
	svcDir := UserSystemdDir()
	if err := os.MkdirAll(svcDir, 0755); err != nil {
		return fmt.Errorf("création %s: %w", svcDir, err)
	}
	svcDest := filepath.Join(svcDir, "userproxyportal.service")
	if err := os.WriteFile(svcDest, serviceContent, 0644); err != nil {
		return fmt.Errorf("écriture service: %w", err)
	}

	return DaemonReload()
}

// Uninstall supprime le binaire (root) et le service utilisateur.
func Uninstall() error {
	// Suppression du binaire via pkexec
	script := fmt.Sprintf("rm -f '%s/userproxyportal'", BinDir)
	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("désinstallation: %w\n%s", err, out)
	}

	// Suppression du service (utilisateur, pas besoin de root)
	svcPath := filepath.Join(UserSystemdDir(), "userproxyportal.service")
	os.Remove(svcPath)

	return nil
}

// WriteConfig écrit config.yaml dans /etc/userproxyportal/ via pkexec.
func WriteConfig(content, certPath string) error {
	cfgTmp, err := writeTempFile("*.yaml", []byte(content))
	if err != nil {
		return err
	}
	defer os.Remove(cfgTmp)

	var cmds []string
	cmds = append(cmds, fmt.Sprintf("mkdir -p '%s'", ConfDir))
	cmds = append(cmds, fmt.Sprintf("cp '%s' '%s/config.yaml' && chmod 644 '%s/config.yaml'", cfgTmp, ConfDir, ConfDir))

	if certPath != "" {
		name := filepath.Base(certPath)
		if ext := filepath.Ext(name); !strings.EqualFold(ext, ".crt") {
			name = name[:len(name)-len(ext)] + ".crt"
		}
		cmds = append(cmds,
			fmt.Sprintf("cp '%s' '/usr/local/share/ca-certificates/%s'", certPath, name),
			"update-ca-certificates",
		)
	}

	script := "set -e\n" + strings.Join(cmds, "\n")
	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("écriture config: %w\n%s", err, out)
	}
	return nil
}

type ServiceState struct {
	Active  bool
	Enabled bool
}

func GetServiceState() ServiceState {
	out, err := exec.Command("systemctl", "--user", "show", "userproxyportal.service",
		"--property=ActiveState,UnitFileState", "--no-pager").Output()
	if err != nil {
		return ServiceState{}
	}
	var s ServiceState
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "ActiveState=") {
			s.Active = strings.TrimPrefix(line, "ActiveState=") == "active"
		}
		if strings.HasPrefix(line, "UnitFileState=") {
			v := strings.TrimPrefix(line, "UnitFileState=")
			s.Enabled = v == "enabled" || v == "enabled-runtime"
		}
	}
	return s
}

func ServiceControl(action string) error {
	out, err := exec.Command("systemctl", "--user", action, "userproxyportal.service").CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w\n%s", action, err, out)
	}
	return nil
}

func DaemonReload() error {
	// S'assurer que XDG_RUNTIME_DIR est défini pour que systemctl --user fonctionne
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		uid := os.Getuid()
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", uid))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("daemon-reload: %w\n%s", err, out)
	}
	return nil
}

func writeTempFile(pattern string, content []byte) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}
