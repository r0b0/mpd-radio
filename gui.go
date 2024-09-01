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

func main() {
    ctx := context.Background()
    fyneApp := app.New()
    w := fyneApp.NewWindow("MPD Radio")
    w.Resize(fyne.NewSize(640, 480))
    application, err := loadApp()
    if err != nil {
        dialog.ShowError(err, w)
        application = &Application{}
    }

    serverLabel := widget.NewLabel("Player")
    var serverDropdown *widget.Select
    serverDropdown = widget.NewSelect(
        application.showServerList(), func(s string) {
            switch s {
            case "Add New...":
                addNewPlayer(ctx, w, func(player *MpdClient) {
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

    w.Show()

    for _, c := range application.ServerList {
        err = c.Connect(ctx)
        if err != nil {
            dialog.ShowError(
                fmt.Errorf("failed to connect to player %s: %w",
                    c.Address, err),
                w)
        } else {
            if serverDropdown.Selected == "" {
                serverDropdown.SetSelected(c.Address)
            }
        }
    }

    fyneApp.Run()
}

func addNewPlayer(ctx context.Context, parent fyne.Window, cb func(client *MpdClient)) {
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
