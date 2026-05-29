package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wisper/userproxyportal/internal/config"
	"github.com/wisper/userproxyportal/internal/proxy"
)

func runApply() {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	cfgPath := fs.String("config", config.DefaultPath, "Chemin vers le fichier de configuration")
	clearMode := fs.Bool("clear", false, "Supprimer la configuration proxy")
	privileged := fs.Bool("privileged", false, "Appliquer /etc/environment et certificat (pkexec)")
	fs.Parse(os.Args[2:])

	if *clearMode {
		if err := proxy.ClearAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration proxy supprimée")
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur config: %v\n", err)
		os.Exit(1)
	}

	if *privileged {
		if err := proxy.ApplyPrivileged(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur privilégiée: %v\n", err)
			os.Exit(1)
		}
	}

	if err := proxy.Apply(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur GNOME: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration proxy appliquée")
}
