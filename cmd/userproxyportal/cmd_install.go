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
	fmt.Printf("✓ Service installé dans %s\n", installer.UserSystemdDir())
	fmt.Println("✓ Répertoire de configuration créé dans /etc/userproxyportal/")
	fmt.Println("✓ daemon-reload effectué")
	fmt.Println()
	fmt.Println("Étapes suivantes:")
	fmt.Println("  1. Configurez le proxy:    userproxyportal setup")
	fmt.Println("  2. Activez le service:     systemctl --user enable --now userproxyportal.service")
	fmt.Println("  3. Entrez vos credentials: userproxyportal")
}
