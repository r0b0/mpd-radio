package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"slices"
)

const AppId = "sk.hq.r0b0.mpdradio"

func main() {
	commandFlag := flag.Bool("command", false, "Show a MPD command window")
	flag.Parse()

	fyneApp := app.NewWithID(AppId)
	width := fyne.NewSize(640, 0)
	a, err := loadApp(fyneApp)
	if err != nil {
		fmt.Printf("failed to load app: %s\n", err)
		a = &Application{fyneApp: fyneApp}
	}
	a.ctx = context.Background()
	a.fyneParent = fyneApp.NewWindow("MPD Radio")
	a.fyneParent.Resize(width)

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
	playerRemoveButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), a.removePlayer)
	playerBox := container.NewBorder(nil, nil, nil, playerRemoveButton, a.playerDropdown)

	radioLabel := widget.NewLabel("Radio")
	a.radioDropdown = widget.NewSelect(
		a.showRadioList(), func(s string) {
			switch s {
			case "Add New...":
				a.addNewRadio()
			}
		})
	radioRemoveButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), a.removeRadio)
	radioBox := container.NewBorder(nil, nil, nil, radioRemoveButton, a.radioDropdown)

	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		err = a.play()
		if err != nil {
			a.ShowError(err)
		}
		a.showPlayerStatus()
	})
	stopButton := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		err = a.stop()
		if err != nil {
			a.ShowError(err)
		}
		a.showPlayerStatus()
	})
	pauseButton := widget.NewButtonWithIcon("Pause", theme.MediaPauseIcon(), func() {
		err = a.pause()
		if err != nil {
			a.ShowError(err)
		}
		a.showPlayerStatus()
	})
	buttonsBox := container.NewHBox(playButton, stopButton, pauseButton)

	a.statusIcon = widget.NewActivity()
	a.statusLabel = widget.NewLabel("")
	statusBox := container.NewBorder(nil, nil, a.statusIcon, nil, a.statusLabel)

	if *commandFlag {
		commandLabel := widget.NewLabel("Command")
		commandEntry := widget.NewEntry()
		commandEntry.OnSubmitted = func(s string) {
			_, playerSelected, err := a.selectedPlayer()
			if err != nil {
				a.ShowError(err)
				return
			}
			resp, err := playerSelected.Command(commandEntry.Text)
			if err != nil {
				a.ShowError(err)
				return
			}
			resp.Print()
			a.displayMpdData(&resp)
		}

		a.fyneParent.SetContent(container.NewVBox(
			playerLabel, playerBox,
			radioLabel, radioBox,
			buttonsBox,
			statusBox,
			commandLabel, commandEntry))
	} else {
		a.fyneParent.SetContent(container.NewVBox(
			playerLabel, playerBox,
			radioLabel, radioBox,
			buttonsBox,
			statusBox))
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
		a.ShowError(
			fmt.Errorf("failed to connect to player %s: %s",
				player.Address, err))
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
					a.ShowError(err)
					return
				}
				a.PlayerList = append(a.PlayerList, player)
				a.playerDropdown.SetOptions(a.showPlayerList())
				a.playerDropdown.SetSelected(player.Address)
				err = a.store()
				if err != nil {
					a.ShowError(err)
				}
			}
		},
		a.fyneParent)
}

func (a *Application) removePlayer() {
	index, player, err := a.selectedPlayer()
	if err != nil {
		return
	}
	dialog.ShowConfirm("Remove player",
		fmt.Sprintf("Are you sure to remove player %s", player.Address),
		func(b bool) {
			if !b {
				return
			}
			a.PlayerList = slices.Delete(a.PlayerList, index, index+1)
			a.playerDropdown.SetOptions(a.showPlayerList())
			err := a.store()
			if err != nil {
				a.ShowError(err)
			}
			a.playerDropdown.ClearSelected()
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
					a.ShowError(err)
				}
			}
		},
		a.fyneParent)
}

func (a *Application) removeRadio() {
	index, radio, err := a.selectedRadio()
	if err != nil {
		return
	}
	dialog.ShowConfirm("Remove radio",
		fmt.Sprintf("Are you sure to remove radio %s", radio.Name),
		func(b bool) {
			if !b {
				return
			}
			a.RadioList = slices.Delete(a.RadioList, index, index+1)
			a.radioDropdown.SetOptions(a.showRadioList())
			err := a.store()
			if err != nil {
				a.ShowError(err)
			}
			a.radioDropdown.ClearSelected()
		},
		a.fyneParent)
}

func (a *Application) showPlayerStatus() {
	status, active, err := a.getPlayerStatus()
	if err != nil {
		if errors.Is(err, &NotFoundError{}) {
			a.statusLabel.SetText("")
			return
		}
		a.ShowError(err)
		a.statusLabel.SetText("")
		return
	}
	a.statusLabel.SetText(status)
	if active {
		a.statusIcon.Start()
	} else {
		a.statusIcon.Stop()
	}
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

func (a *Application) ShowError(err error) {
	dialog.ShowError(err, a.fyneParent)
}
