package main

import (
	"context"
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const AppId = "sk.hq.r0b0.mpdradio"

func main() {
	commandFlag := flag.Bool("Command", true, "Show a Command windows")
	flag.Parse()

	fyneApp := app.NewWithID(AppId)
	a, err := loadApp(fyneApp)
	if err != nil {
		fmt.Printf("failed to load app: %s\n", err)
		a = &Application{fyneApp: fyneApp}
	}
	a.ctx = context.Background()
	a.fyneParent = fyneApp.NewWindow("MPD Radio")
	a.fyneParent.Resize(fyne.NewSize(640, 0))

	playerLabel := widget.NewLabel("Player")
	a.playerDropdown = widget.NewSelect(
		a.showPlayerList(), func(s string) {
			switch s {
			case "Add New...":
				go a.addNewPlayer()
			default:
				go a.showPlayerStatus()
			}
		})

	radioLabel := widget.NewLabel("Radio")
	a.radioDropdown = widget.NewSelect(
		a.showRadioList(), func(s string) {
			switch s {
			case "Add New...":
				a.addNewRadio()
			}
		})

	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		err = a.play()
		if err != nil {
			dialog.ShowError(err, a.fyneParent)
		}
		a.showPlayerStatus()
	})
	stopButton := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		err = a.stop()
		if err != nil {
			dialog.ShowError(err, a.fyneParent)
		}
		a.showPlayerStatus()
	})
	pauseButton := widget.NewButtonWithIcon("Pause", theme.MediaPauseIcon(), func() {
		err = a.pause()
		if err != nil {
			dialog.ShowError(err, a.fyneParent)
		}
		a.showPlayerStatus()
	})
	buttonsBox := container.NewHBox(playButton, stopButton, pauseButton)

	a.statusLabel = widget.NewLabel("")

	if *commandFlag {
		commandLabel := widget.NewLabel("Command")
		commandEntry := widget.NewEntry()
		commandEntry.OnSubmitted = func(s string) {
			playerSelected, err := a.selectedPlayer()
			if err != nil {
				dialog.ShowError(err, a.fyneParent)
				return
			}
			resp, err := playerSelected.Command(commandEntry.Text)
			if err != nil {
				dialog.ShowError(err, a.fyneParent)
				return
			}
			resp.Print()
			a.displayMpdData(&resp)
		}

		a.fyneParent.SetContent(container.NewVBox(
			playerLabel, a.playerDropdown,
			radioLabel, a.radioDropdown,
			buttonsBox,
			a.statusLabel,
			commandLabel, commandEntry))
	} else {
		a.fyneParent.SetContent(container.NewVBox(
			playerLabel, a.playerDropdown,
			radioLabel, a.radioDropdown,
			buttonsBox,
			a.statusLabel))
	}

	a.fyneParent.Show()

	for _, p := range a.PlayerList {
		go a.connectPlayer(p)
	}

	fyneApp.Run()
}

func (a *Application) connectPlayer(player *MpdClient) {
	err := player.Connect(a.ctx)
	if err != nil {
		dialog.ShowError(
			fmt.Errorf("failed to connect to player %s: %s",
				player.Address, err),
			a.fyneParent)
	} else {
		if a.playerDropdown.Selected == "" {
			a.playerDropdown.SetSelected(player.Address)
		}
	}
}

func (a *Application) addNewPlayer() {
	hostEntry := widget.NewEntry()
	portEntry := widget.NewEntry()
	portEntry.SetText("6600")
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
				player, err := NewMpdClient(a.ctx, hostEntry.Text, portEntry.Text)
				if err != nil {
					dialog.ShowError(err, a.fyneParent)
					return
				}
				a.PlayerList = append(a.PlayerList, player)
				a.playerDropdown.SetOptions(a.showPlayerList())
				a.playerDropdown.SetSelected(player.Address)
				err = a.store()
				if err != nil {
					dialog.ShowError(err, a.fyneParent)
				}
			}
		},
		a.fyneParent)
}

type Radio struct {
	Name string
	Url  string
}

func (a *Application) addNewRadio() {
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
				radio := Radio{nameEntry.Text, urlEntry.Text}
				a.RadioList = append(a.RadioList, radio)
				a.radioDropdown.SetOptions(a.showRadioList())
				a.radioDropdown.SetSelected(radio.Name)
				err := a.store()
				if err != nil {
					dialog.ShowError(err, a.fyneParent)
				}
			}
		},
		a.fyneParent)
}

func (a *Application) showPlayerStatus() {
	status, err := a.getPlayerStatus()
	if err != nil {
		dialog.ShowError(err, a.fyneParent)
		a.statusLabel.SetText("")
		return
	}
	a.statusLabel.SetText(status)
}

func (a *Application) displayMpdData(d *MpdData) {
	grid := layout.NewGridLayout(2)
	cont := container.New(grid,
		widget.NewLabel("Command"), widget.NewLabel(d.Command))

	for k, v := range d.Response {
		cont.Add(widget.NewLabel(k))
		cont.Add(widget.NewLabel(v))
	}
	cont.Add(widget.NewLabel("OK"))
	cont.Add(widget.NewLabel(d.Ok))

	dialog.ShowCustom("Data from player", "OK", cont, a.fyneParent)
}
