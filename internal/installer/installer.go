package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	BinDir     = "/usr/local/bin"
	ConfDir    = "/etc/userproxyportal"
	SystemdDir = "/etc/systemd/user"
)

type ComponentStatus struct {
	Name      string
	Installed bool
	Path      string
}

func CheckComponents() []ComponentStatus {
	items := []struct{ name, path string }{
		{"userproxyportal", BinDir + "/userproxyportal"},
		{"userproxyportal.service", SystemdDir + "/userproxyportal.service"},
		{"config.yaml", ConfDir + "/config.yaml"},
	}
	statuses := make([]ComponentStatus, len(items))
	for i, it := range items {
		_, err := os.Stat(it.path)
		statuses[i] = ComponentStatus{Name: it.name, Path: it.path, Installed: err == nil}
	}
	return statuses
}

// SelfInstall copies the running executable and embedded assets via pkexec.
func SelfInstall(exePath string, serviceContent, configContent []byte) error {
	svcTmp, err := writeTempFile("*.service", serviceContent)
	if err != nil {
		return err
	}
	defer os.Remove(svcTmp)

	cfgTmp, err := writeTempFile("*.yaml", configContent)
	if err != nil {
		return err
	}
	defer os.Remove(cfgTmp)

	binDest := BinDir + "/userproxyportal"
	svcDest := SystemdDir + "/userproxyportal.service"
	cfgDest := ConfDir + "/config.yaml"

	script := fmt.Sprintf(`set -e
cp '%s' '%s' && chmod 755 '%s'
mkdir -p '%s' && cp '%s' '%s'
mkdir -p '%s'
[ -f '%s' ] || cp '%s' '%s'
`, exePath, binDest, binDest,
		SystemdDir, svcTmp, svcDest,
		ConfDir, cfgDest, cfgTmp, cfgDest)

	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation: %w\n%s", err, out)
	}
	return nil
}

// Uninstall removes the binary and the service file via pkexec.
func Uninstall() error {
	script := fmt.Sprintf(`set -e
rm -f '%s/userproxyportal'
rm -f '%s/userproxyportal.service'
`, BinDir, SystemdDir)
	out, err := exec.Command("pkexec", "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("désinstallation: %w\n%s", err, out)
	}
	return nil
}

// WriteConfig writes config content to /etc/userproxyportal/config.yaml via pkexec.
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

// ServiceState holds the current state of the systemd user service.
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

// DaemonReload demande au daemon systemd utilisateur de relire ses unités.
// À appeler après toute installation ou modification de fichier .service.
func DaemonReload() error {
	out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput()
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
