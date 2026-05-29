package main

import (
	"fmt"
	"os"
)

// version est injectée au build via -ldflags="-X main.version=..."
var version = "dev"

func main() {
	subcmd := ""
	if len(os.Args) > 1 {
		subcmd = os.Args[1]
	}

	switch subcmd {
	case "proxy":
		runProxy()
	case "apply":
		runApply()
	case "install":
		runInstall()
	case "setup":
		runSetup()
	case "version", "--version", "-v":
		fmt.Println("userproxyportal", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		runGUI()
	}
}

func printHelp() {
	fmt.Print(`userproxyportal — Gestionnaire de proxy d'entreprise

Usage:
  userproxyportal              Interface utilisateur (identifiants AD)
  userproxyportal setup        Interface d'administration
  userproxyportal proxy        Démarrer le daemon proxy local (systemd)
  userproxyportal apply        Appliquer la configuration proxy
  userproxyportal install      Installer le programme dans /usr/local/bin
`)
}
