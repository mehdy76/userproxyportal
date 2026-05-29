package main

import (
	"fmt"
	"os"

	"github.com/wisper/userproxyportal/internal/installer"
)

func runInstall() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Impossible de déterminer le chemin du binaire: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Installation de User Proxy Portal...")

	if err := installer.SelfInstall(exe, embeddedService, embeddedConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Binaire installé dans /usr/local/bin/userproxyportal")
	fmt.Println("✓ Service systemd installé dans /etc/systemd/user/")
	fmt.Println("✓ Répertoire de configuration créé dans /etc/userproxyportal/")

	fmt.Print("→ Rechargement du daemon systemd utilisateur... ")
	if err := installer.DaemonReload(); err != nil {
		fmt.Fprintf(os.Stderr, "\n⚠ daemon-reload: %v\n", err)
		fmt.Println("  Lancez manuellement: systemctl --user daemon-reload")
	} else {
		fmt.Println("OK")
	}

	fmt.Println()
	fmt.Println("Étapes suivantes:")
	fmt.Println("  1. Configurez le proxy:    userproxyportal setup")
	fmt.Println("  2. Activez le service:     systemctl --user enable --now userproxyportal.service")
	fmt.Println("  3. Entrez vos credentials: userproxyportal")
}
