package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("MPD Radio")
	w.Resize(fyne.NewSize(640, 480))

	serverLabel := widget.NewLabel("Player")
	serverList := []MpdClient{}
	var serverDropdown *widget.Select
	serverDropdown = widget.NewSelect(showServerList(serverList), func(s string) {
		switch s {
		case "Add New...":
			addNewPlayer(w, func(player MpdClient) {
				serverList = append(serverList, player)
				serverDropdown.SetOptions(showServerList(serverList))
			})
		}
	})

	urlLabel := widget.NewLabel("Radio")
	urlList := []Radio{}
	var urlDropdown *widget.Select
	urlDropdown = widget.NewSelect(showRadioList(urlList), func(s string) {
		switch s {
		case "Add New...":
			addNewRadio(w, func(radio Radio) {
				urlList = append(urlList, radio)
				urlDropdown.SetOptions(showRadioList(urlList))
			})
		}
	})

	// TODO buttons
	w.SetContent(container.NewVBox(
		serverLabel, serverDropdown, urlLabel, urlDropdown))
	w.ShowAndRun()
}

func showServerList(servers []MpdClient) []string {
	ret := []string{"Add New..."}
	for _, s := range servers {
		ret = append(ret, s.address)
	}
	return ret
}

func showRadioList(radios []Radio) []string {
	ret := []string{"Add New..."}
	for _, s := range radios {
		ret = append(ret, s.name)
	}
	return ret
}

func addNewPlayer(parent fyne.Window, cb func(client MpdClient)) {
	hostEntry := widget.NewEntry()
	portEntry := widget.NewEntry()
	formItems := []*widget.FormItem{
		widget.NewFormItem("Host", hostEntry),
		widget.NewFormItem("Port", portEntry),
	}
	dialog.ShowForm("Add new player",
		"OK",
		"Cancel",
		formItems,
		func(b bool) {
			if b {
				client, err := Connect(hostEntry.Text, portEntry.Text)
				if err != nil {
					dialog.ShowError(err, parent)
					return
				}
				cb(client)
			}
		},
		parent)
}

type Radio struct {
	name string
	url  string
}

func addNewRadio(parent fyne.Window, cb func(Radio)) {
	nameEntry := widget.NewEntry()
	urlEntry := widget.NewEntry()
	formItems := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("URL", urlEntry),
	}
	dialog.ShowForm("Add new radio",
		"OK",
		"Cancel",
		formItems,
		func(b bool) {
			if b {
				cb(Radio{nameEntry.Text, urlEntry.Text})
			}
		},
		parent)
}
