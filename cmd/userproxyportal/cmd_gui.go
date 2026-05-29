package main

import (
	"fmt"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/wisper/userproxyportal/internal/config"
	"github.com/wisper/userproxyportal/internal/keyring"
	"github.com/wisper/userproxyportal/internal/proxy"
)

func runGUI() {
	a := app.New()
	w := a.NewWindow("User Proxy Portal")
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(420, 300))

	cfg, cfgErr := config.Load(config.DefaultPath)

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Nom d'utilisateur AD")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Mot de passe")

	statusLabel := widget.NewLabel("En attente")
	statusLabel.Wrapping = fyne.TextWrapWord

	if existingUser, _, err := keyring.Load(); err == nil && existingUser != "" {
		usernameEntry.SetText(existingUser)
		statusLabel.SetText("Identifiants existants chargés")
	}

	setStatus := func(msg string, isErr bool) {
		statusLabel.SetText(msg)
		if isErr {
			statusLabel.Importance = widget.DangerImportance
		} else {
			statusLabel.Importance = widget.SuccessImportance
		}
		statusLabel.Refresh()
	}

	applyBtn := widget.NewButtonWithIcon("Appliquer", theme.ConfirmIcon(), func() {
		if cfgErr != nil {
			dialog.ShowError(fmt.Errorf("configuration manquante:\n%v\n\nLancez: userproxyportal setup", cfgErr), w)
			return
		}
		username := usernameEntry.Text
		password := passwordEntry.Text
		if username == "" || password == "" {
			setStatus("Veuillez renseigner les identifiants", true)
			return
		}
		setStatus("Application en cours...", false)
		go func() {
			if err := keyring.Save(username, password); err != nil {
				setStatus(fmt.Sprintf("Erreur keyring: %v", err), true)
				return
			}
			if err := proxy.ApplyPrivileged(cfg); err != nil {
				setStatus(fmt.Sprintf("Erreur (privilèges): %v", err), true)
				return
			}
			if err := proxy.Apply(cfg); err != nil {
				setStatus(fmt.Sprintf("Erreur GNOME: %v", err), true)
				return
			}
			if err := exec.Command("systemctl", "--user", "restart", "userproxyportal.service").Run(); err != nil {
				setStatus(fmt.Sprintf("Proxy configuré (service: %v)", err), false)
				return
			}
			setStatus("Proxy configuré avec succès", false)
		}()
	})
	applyBtn.Importance = widget.HighImportance

	clearBtn := widget.NewButtonWithIcon("Désactiver", theme.CancelIcon(), func() {
		go func() {
			exec.Command("systemctl", "--user", "stop", "userproxyportal.service").Run()
			if err := proxy.ClearAll(); err != nil {
				setStatus(fmt.Sprintf("Erreur désactivation: %v", err), true)
				return
			}
			keyring.Delete()
			setStatus("Proxy désactivé", false)
		}()
	})

	proxyInfo := widget.NewLabel("")
	proxyInfo.Importance = widget.LowImportance
	if cfgErr == nil {
		proxyInfo.SetText(fmt.Sprintf("Proxy: %s:%d  →  local: 127.0.0.1:%d",
			cfg.Proxy.Host, cfg.Proxy.Port, cfg.Proxy.GetLocalPort()))
	} else {
		proxyInfo.SetText("⚠ Configuration manquante — lancez: userproxyportal setup")
		proxyInfo.Importance = widget.DangerImportance
	}

	w.SetContent(container.NewVBox(
		container.NewPadded(proxyInfo),
		widget.NewSeparator(),
		container.NewPadded(container.NewVBox(
			widget.NewLabel("Identifiants Active Directory"),
			usernameEntry,
			passwordEntry,
		)),
		container.NewPadded(container.NewHBox(applyBtn, clearBtn)),
		widget.NewSeparator(),
		container.NewPadded(statusLabel),
	))
	w.ShowAndRun()
}
