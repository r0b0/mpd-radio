package main

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Application struct {
	ServerList     []*MpdClient
	SelectedServer int // TODO
	UrlList        []Radio
	SelectedUrl    int // TODO
}

func main() {
	a := app.New()
	w := a.NewWindow("MPD Radio")
	w.Resize(fyne.NewSize(640, 480))
	application, err := loadApp()
	if err != nil {
		dialog.ShowError(err, w)
		return
	}

	serverLabel := widget.NewLabel("Player")
	var serverDropdown *widget.Select
	serverDropdown = widget.NewSelect(
		application.showServerList(), func(s string) {
			switch s {
			case "Add New...":
				addNewPlayer(w, func(player *MpdClient) {
					application.ServerList = append(application.ServerList, player)
					serverDropdown.SetOptions(application.showServerList())
					serverDropdown.SetSelected(player.Address)
					err := application.store()
					if err != nil {
						dialog.ShowError(err, w)
					}
				})
			}
		})

	urlLabel := widget.NewLabel("Radio")
	var urlDropdown *widget.Select
	urlDropdown = widget.NewSelect(
		application.showRadioList(), func(s string) {
			switch s {
			case "Add New...":
				addNewRadio(w, func(radio Radio) {
					application.UrlList = append(application.UrlList, radio)
					urlDropdown.SetOptions(application.showRadioList())
					urlDropdown.SetSelected(radio.Name)
					err := application.store()
					if err != nil {
						dialog.ShowError(err, w)
					}
				})
			}
		})

	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		fmt.Printf("Play button pressed\n")
		// TODO
	})
	stopButton := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		fmt.Printf("Stop button pressed\n")
		// TODO
	})
	pauseButton := widget.NewButtonWithIcon("Pause", theme.MediaPauseIcon(), func() {
		fmt.Printf("Pause button pressed\n")
		// TODO
	})
	buttonsBox := container.NewHBox(playButton, stopButton, pauseButton)

	commandLabel := widget.NewLabel("Command")
	commandEntry := widget.NewEntry()
	commandEntry.OnSubmitted = func(s string) {
		serverSelected, err := application.selectedServer(serverDropdown.Selected)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		fmt.Printf("Command executed: %s for %v\n", commandEntry.Text, serverSelected)
		resp, err := serverSelected.Command(commandEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		resp.Print()
	}

	w.SetContent(container.NewVBox(
		serverLabel, serverDropdown,
		urlLabel, urlDropdown,
		buttonsBox,
		commandLabel, commandEntry))
	w.ShowAndRun()
}

func (a *Application) store() error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	err = saveConfig(string(data))
	if err != nil {
		return err
	}
	return nil
}

func loadApp() (*Application, error) {
	data, err := loadConfig()
	if err != nil {
		return nil, err
	}
	var application Application
	err = json.Unmarshal([]byte(data), &application)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (a *Application) showServerList() []string {
	ret := []string{"Add New..."}
	for _, s := range a.ServerList {
		ret = append(ret, s.Address)
	}
	return ret
}

func (a *Application) selectedServer(name string) (*MpdClient, error) {
	for _, s := range a.ServerList {
		if s.Address == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("server not found")
}

func (a *Application) showRadioList() []string {
	ret := []string{"Add New..."}
	for _, s := range a.UrlList {
		ret = append(ret, s.Name)
	}
	return ret
}

func (a *Application) selectedRadio(name string) (*Radio, error) {
	for _, r := range a.UrlList {
		if r.Name == name {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("radio not found")
}

func addNewPlayer(parent fyne.Window, cb func(client *MpdClient)) {
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
				client, err := NewMpdClient(hostEntry.Text, portEntry.Text)
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
	Name string
	Url  string
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
