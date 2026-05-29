package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

func runSetup() {
	a := app.New()
	w := a.NewWindow("User Proxy Portal — Administration")
	w.Resize(fyne.NewSize(580, 500))

	tabs := container.NewAppTabs(
		container.NewTabItem("Installation", makeInstallTab(w)),
		container.NewTabItem("Configuration", makeConfigTab(w)),
		container.NewTabItem("Service", makeServiceTab(w)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.ShowAndRun()
}
