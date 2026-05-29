package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/wisper/userproxyportal/internal/installer"
)

func makeServiceTab(_ fyne.Window) fyne.CanvasObject {
	activeLabel := widget.NewLabel("")
	enabledLabel := widget.NewLabel("")
	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord

	setStatus := func(msg string, isErr bool) {
		statusLabel.SetText(msg)
		if isErr {
			statusLabel.Importance = widget.DangerImportance
		} else {
			statusLabel.Importance = widget.SuccessImportance
		}
		statusLabel.Refresh()
	}

	refresh := func() {
		state := installer.GetServiceState()
		if state.Active {
			activeLabel.SetText("● En cours d'exécution")
			activeLabel.Importance = widget.SuccessImportance
		} else {
			activeLabel.SetText("○ Arrêté")
			activeLabel.Importance = widget.DangerImportance
		}
		if state.Enabled {
			enabledLabel.SetText("↺ Démarrage automatique activé")
			enabledLabel.Importance = widget.SuccessImportance
		} else {
			enabledLabel.SetText("— Démarrage automatique désactivé")
			enabledLabel.Importance = widget.LowImportance
		}
		activeLabel.Refresh()
		enabledLabel.Refresh()
	}
	refresh()

	action := func(cmd string) {
		go func() {
			if err := installer.ServiceControl(cmd); err != nil {
				setStatus(fmt.Sprintf("Erreur: %v", err), true)
				return
			}
			refresh()
			setStatus(fmt.Sprintf("systemctl --user %s: OK", cmd), false)
		}()
	}

	startBtn := widget.NewButtonWithIcon("Démarrer", theme.MediaPlayIcon(), func() { action("start") })
	stopBtn := widget.NewButtonWithIcon("Arrêter", theme.MediaStopIcon(), func() { action("stop") })
	restartBtn := widget.NewButtonWithIcon("Redémarrer", theme.ViewRefreshIcon(), func() { action("restart") })
	startBtn.Importance = widget.HighImportance

	enableBtn := widget.NewButtonWithIcon("Activer au démarrage", theme.ConfirmIcon(), func() { action("enable") })
	disableBtn := widget.NewButtonWithIcon("Désactiver", theme.CancelIcon(), func() { action("disable") })

	refreshBtn := widget.NewButtonWithIcon("Actualiser", theme.ViewRefreshIcon(), func() {
		refresh()
		setStatus("Statut actualisé", false)
	})

	return container.NewVBox(
		widget.NewCard("Statut", "", container.NewVBox(activeLabel, enabledLabel)),
		widget.NewCard("Contrôles", "", container.NewVBox(
			container.NewHBox(startBtn, stopBtn, restartBtn),
			container.NewHBox(enableBtn, disableBtn),
		)),
		container.NewPadded(container.NewHBox(refreshBtn)),
		container.NewPadded(statusLabel),
	)
}
