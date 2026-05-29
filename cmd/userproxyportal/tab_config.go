package main

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"

	"github.com/wisper/userproxyportal/internal/config"
	"github.com/wisper/userproxyportal/internal/installer"
)

func makeConfigTab(w fyne.Window) fyne.CanvasObject {
	proxyHost := widget.NewEntry()
	proxyHost.SetPlaceHolder("proxy.entreprise.local")

	proxyPort := widget.NewEntry()
	proxyPort.SetPlaceHolder("8080")

	localPort := widget.NewEntry()
	localPort.SetPlaceHolder("3128")

	noProxy := widget.NewEntry()
	noProxy.SetPlaceHolder("localhost,127.0.0.1,::1,.entreprise.local")

	pacURL := widget.NewEntry()
	pacURL.SetPlaceHolder("https://proxy.entreprise.local/proxy.pac (optionnel)")

	certPath := widget.NewEntry()
	certPath.SetPlaceHolder("/etc/userproxyportal/entreprise-ca.cer")

	installCert := widget.NewCheck("Installer le certificat dans le trust store système", nil)
	installCert.SetChecked(true)

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

	if cfg, err := config.Load(config.DefaultPath); err == nil {
		proxyHost.SetText(cfg.Proxy.Host)
		proxyPort.SetText(strconv.Itoa(cfg.Proxy.Port))
		if cfg.Proxy.LocalPort > 0 {
			localPort.SetText(strconv.Itoa(cfg.Proxy.LocalPort))
		}
		noProxy.SetText(cfg.Proxy.NoProxy)
		pacURL.SetText(cfg.Proxy.PACUrl)
		certPath.SetText(cfg.Certificate.Path)
	}

	certBrowseBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil || uri == nil {
				return
			}
			uri.Close()
			certPath.SetText(uri.URI().Path())
		}, w)
	})

	saveBtn := widget.NewButtonWithIcon("Sauvegarder la configuration", theme.DocumentSaveIcon(), func() {
		port, err := strconv.Atoi(proxyPort.Text)
		if err != nil || port <= 0 {
			setStatus("Port proxy invalide", true)
			return
		}
		lport, err := strconv.Atoi(localPort.Text)
		if err != nil || lport <= 0 {
			setStatus("Port local invalide", true)
			return
		}
		if proxyHost.Text == "" {
			setStatus("Hôte proxy requis", true)
			return
		}

		cfg := config.Config{
			Proxy: config.ProxyConfig{
				Host:      proxyHost.Text,
				Port:      port,
				LocalPort: lport,
				NoProxy:   noProxy.Text,
				PACUrl:    pacURL.Text,
			},
			Certificate: config.CertificateConfig{
				Path: certPath.Text,
			},
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			setStatus(fmt.Sprintf("Erreur sérialisation: %v", err), true)
			return
		}

		setStatus("Sauvegarde en cours...", false)
		go func() {
			certToInstall := ""
			if installCert.Checked && certPath.Text != "" {
				certToInstall = certPath.Text
			}
			if err := installer.WriteConfig(string(data), certToInstall); err != nil {
				setStatus(err.Error(), true)
				return
			}
			setStatus("Configuration sauvegardée", false)
		}()
	})
	saveBtn.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("Hôte proxy", proxyHost),
		widget.NewFormItem("Port proxy", proxyPort),
		widget.NewFormItem("Port local", localPort),
		widget.NewFormItem("Exclusions (no_proxy)", noProxy),
		widget.NewFormItem("URL PAC", pacURL),
		widget.NewFormItem("Certificat SSL", container.NewBorder(nil, nil, nil, certBrowseBtn, certPath)),
		widget.NewFormItem("", installCert),
	)

	return container.NewVBox(
		container.NewPadded(form),
		container.NewPadded(saveBtn),
		container.NewPadded(statusLabel),
	)
}
