package main

import (
    "context"
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

const AppId = "sk.hq.r0b0.mpd-radio"

func main() {
    ctx := context.Background()
    fyneApp := app.NewWithID(AppId)
    fyneWindow := fyneApp.NewWindow("MPD Radio")
    fyneWindow.Resize(fyne.NewSize(640, 480))
    application, err := loadApp(fyneApp)
    if err != nil {
        dialog.ShowError(err, fyneWindow)
        application = &Application{fyneApp: fyneApp}
    }

    playerLabel := widget.NewLabel("Player")
    var playerDropdown *widget.Select
    playerDropdown = widget.NewSelect(
        application.showPlayerList(), nil)

    radioLabel := widget.NewLabel("Radio")
    var radioDropdown *widget.Select
    radioDropdown = widget.NewSelect(
        application.showRadioList(), func(s string) {
            switch s {
            case "Add New...":
                addNewRadio(fyneWindow, func(radio Radio) {
                    application.RadioList = append(application.RadioList, radio)
                    radioDropdown.SetOptions(application.showRadioList())
                    radioDropdown.SetSelected(radio.Name)
                    err := application.store()
                    if err != nil {
                        dialog.ShowError(err, fyneWindow)
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

    statusLabel := widget.NewLabel("")

    commandLabel := widget.NewLabel("Command")
    commandEntry := widget.NewEntry()
    commandEntry.OnSubmitted = func(s string) {
        playerSelected, err := application.selectedPlayer(playerDropdown.Selected)
        if err != nil {
            dialog.ShowError(err, fyneWindow)
            return
        }
        fmt.Printf("Command executed: %s for %v\n", commandEntry.Text, playerSelected)
        resp, err := playerSelected.Command(commandEntry.Text)
        if err != nil {
            dialog.ShowError(err, fyneWindow)
            return
        }
        resp.Print()
    }

    fyneWindow.SetContent(container.NewVBox(
        playerLabel, playerDropdown,
        radioLabel, radioDropdown,
        buttonsBox,
        statusLabel,
        commandLabel, commandEntry))

    playerDropdown.OnChanged = func(s string) {
        switch s {
        case "Add New...":
            addNewPlayer(ctx, fyneWindow, func(player *MpdClient) {
                application.PlayerList = append(application.PlayerList, player)
                playerDropdown.SetOptions(application.showPlayerList())
                playerDropdown.SetSelected(player.Address)
                err := application.store()
                if err != nil {
                    dialog.ShowError(err, fyneWindow)
                }
            })
        default:
            playerSelected, err := application.selectedPlayer(s)
            if err != nil {
                return
            }
            go func() {
                err := showPlayerStatus(statusLabel, playerSelected)
                if err != nil {
                    dialog.ShowError(fmt.Errorf("failed to show status of player %s: %w",
                                                playerSelected.Address, err),
                                    fyneWindow)
                }
            }()
        }
    }
    fyneWindow.Show()

    for _, c := range application.PlayerList {
        err = c.Connect(ctx)
        if err != nil {
            dialog.ShowError(
                fmt.Errorf("failed to connect to player %s: %fyneWindow",
                    c.Address, err),
                fyneWindow)
        } else {
            if playerDropdown.Selected == "" {
                playerDropdown.SetSelected(c.Address)
            }
        }
    }

    fyneApp.Run()
}

func addNewPlayer(ctx context.Context, parent fyne.Window, cb func(client *MpdClient)) {
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
                client, err := NewMpdClient(ctx, hostEntry.Text, portEntry.Text)
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

func showPlayerStatus(statusLabel *widget.Label, player *MpdClient) error {
    data, err := player.Command("status")
    statusLabel.SetText("")
    if err != nil {
        return err
    }
    data.Print() // TODO
    status, ok := data.response["state"]
    if !ok {
        return fmt.Errorf("failed to get player status")
    }
    switch status {
    case "play":
        songData, err := player.Command("currentsong")
        if err != nil {
            return err
        }
        name, ok := songData.response["Name"]
        if !ok {
            return nil
            // TODO check for other fields?
        }
        statusLabel.SetText(fmt.Sprintf("Playing: %s", name))
    case "stop":
        statusLabel.SetText("Stopped")
    case "pause":
        statusLabel.SetText("Paused")
    }
    return nil
}
