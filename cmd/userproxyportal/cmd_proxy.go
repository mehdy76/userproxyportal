package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wisper/userproxyportal/internal/config"
	"github.com/wisper/userproxyportal/internal/keyring"
	"github.com/wisper/userproxyportal/internal/proxyserver"
)

func runProxy() {
	fs := flag.NewFlagSet("proxy", flag.ExitOnError)
	debug := fs.Bool("debug", false, "Afficher les requêtes en temps réel")
	fs.Parse(os.Args[2:])

	cfg, err := config.Load(config.DefaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur config: %v\n", err)
		os.Exit(1)
	}

	username, password, err := keyring.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Aucun identifiant trouvé. Lancez d'abord: userproxyportal\n")
		os.Exit(1)
	}

	listenAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Proxy.GetLocalPort())
	upstreamAddr := fmt.Sprintf("%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)

	srv := proxyserver.New(listenAddr, upstreamAddr)
	srv.SetCredentials(username, password)
	srv.SetDebug(*debug)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGHUP:
				u, p, err := keyring.Load()
				if err != nil {
					fmt.Fprintf(os.Stderr, "SIGHUP: keyring: %v\n", err)
					continue
				}
				srv.SetCredentials(u, p)
				fmt.Fprintln(os.Stderr, "Credentials rechargés")
			case syscall.SIGTERM, syscall.SIGINT:
				srv.Shutdown(context.Background())
				os.Exit(0)
			}
		}
	}()

	if *debug {
		fmt.Fprintf(os.Stderr, "Mode debug activé\n")
	}
	fmt.Fprintf(os.Stderr, "Proxy: %s → %s\n", listenAddr, upstreamAddr)

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}
}
