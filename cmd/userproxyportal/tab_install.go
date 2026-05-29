package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/wisper/userproxyportal/internal/installer"
)

func makeInstallTab(w fyne.Window) fyne.CanvasObject {
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

	type row struct{ icon, path *widget.Label }
	rows := map[string]*row{}

	updateIcon := func(r *row, installed bool) {
		if installed {
			r.icon.SetText("✓")
			r.icon.Importance = widget.SuccessImportance
		} else {
			r.icon.SetText("✗")
			r.icon.Importance = widget.DangerImportance
		}
		r.icon.Refresh()
	}

	var statusRows []fyne.CanvasObject
	for _, c := range installer.CheckComponents() {
		icon := widget.NewLabel("")
		pathLbl := widget.NewLabel(c.Path)
		pathLbl.Importance = widget.LowImportance
		nameLbl := widget.NewLabel(c.Name)
		nameLbl.TextStyle = fyne.TextStyle{Monospace: true}

		rows[c.Name] = &row{icon: icon, path: pathLbl}
		updateIcon(rows[c.Name], c.Installed)

		statusRows = append(statusRows, container.NewBorder(nil, nil, icon, pathLbl, nameLbl))
	}

	refresh := func() {
		for _, c := range installer.CheckComponents() {
			if r, ok := rows[c.Name]; ok {
				updateIcon(r, c.Installed)
			}
		}
	}

	installBtn := widget.NewButtonWithIcon("Installer / Mettre à jour", theme.DownloadIcon(), func() {
		setStatus("Installation en cours...", false)
		exe, _ := os.Executable()
		go func() {
			if err := installer.SelfInstall(exe, embeddedService, embeddedConfig); err != nil {
				setStatus(err.Error(), true)
				return
			}
			refresh()
			setStatus("Installation réussie", false)
		}()
	})
	installBtn.Importance = widget.HighImportance

	uninstallBtn := widget.NewButtonWithIcon("Désinstaller", theme.DeleteIcon(), func() {
		dialog.ShowConfirm(
			"Confirmer la désinstallation",
			"Supprimer le binaire et le service systemd ?",
			func(ok bool) {
				if !ok {
					return
				}
				go func() {
					installer.ServiceControl("stop")
					installer.ServiceControl("disable")
					if err := installer.Uninstall(); err != nil {
						setStatus(err.Error(), true)
						return
					}
					refresh()
					setStatus("Désinstallation réussie", false)
				}()
			}, w)
	})

	return container.NewVBox(
		widget.NewCard("Composants installés", "", container.NewVBox(statusRows...)),
		container.NewPadded(container.NewHBox(installBtn, uninstallBtn)),
		container.NewPadded(statusLabel),
	)
}
