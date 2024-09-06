package main

import (
	"context"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type Application struct {
	PlayerList     []*MpdClient
	RadioList      []Radio
	fyneApp        fyne.App
	fyneParent     fyne.Window
	playerDropdown *widget.Select
	radioDropdown  *widget.Select
	statusLabel    *widget.Label
	ctx            context.Context
}

func (a *Application) store() error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	a.fyneApp.Preferences().SetString("config", string(data))
	return nil
}

func loadApp(fyneApp fyne.App) (*Application, error) {
	data := fyneApp.Preferences().String("config")
	var application Application
	if data != "" {
		err := json.Unmarshal([]byte(data), &application)
		if err != nil {
			return nil, err
		}
	}
	application.fyneApp = fyneApp
	return &application, nil
}

func (a *Application) showPlayerList() []string {
	ret := []string{"Add New..."}
	for _, s := range a.PlayerList {
		ret = append(ret, s.Address)
	}
	return ret
}

func (a *Application) selectedPlayer() (*MpdClient, error) {
	for _, s := range a.PlayerList {
		if s.Address == a.playerDropdown.Selected {
			return s, nil
		}
	}
	return nil, fmt.Errorf("server not found")
}

func (a *Application) showRadioList() []string {
	ret := []string{"Add New..."}
	for _, s := range a.RadioList {
		ret = append(ret, s.Name)
	}
	return ret
}

func (a *Application) selectedRadio() (*Radio, error) {
	for _, r := range a.RadioList {
		if r.Name == a.radioDropdown.Selected {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("radio not found")
}

func (a *Application) getPlayerStatus() (string, error) {
	player, err := a.selectedPlayer()
	if err != nil {
		return "", err
	}
	data, err := player.Command("status")
	if err != nil {
		return "", err
	}
	data.Print()
	status, ok := data.Response["state"]
	if !ok {
		return "", fmt.Errorf("failed to get player status")
	}
	switch status {
	case "play":
		songData, err := player.Command("currentsong")
		if err != nil {
			return "", err
		}
		songData.Print()
		var playing string
		name, ok := songData.Response["Name"]
		if ok {
			playing = name
			// TODO check for other fields?
		} else {
			playing = songData.Response["file"]
		}
		return fmt.Sprintf("Playing: %s", playing), nil
	case "stop":
		return "Stopped", nil
	case "pause":
		return "Paused", nil
	}
	return "", nil

}

func (a *Application) play() error {
	player, err := a.selectedPlayer()
	if err != nil {
		return err
	}
	radio, err := a.selectedRadio()
	if err != nil {
		return err
	}
	_, err = player.Command("clear")
	if err != nil {
		return err
	}
	addIdData, err := player.Command(fmt.Sprintf("addid \"%s\" 0", radio.Url))
	if err != nil {
		return err
	}
	addIdData.Print()
	id, ok := addIdData.Response["Id"]
	if !ok {
		return fmt.Errorf("failed to get id of added song")
	}
	_, err = player.Command(fmt.Sprintf("playid %s", id))
	if err != nil {
		return err
	}
	return nil
}

func (a *Application) stop() error {
	player, err := a.selectedPlayer()
	if err != nil {
		return err
	}
	_, err = player.Command("stop")
	if err != nil {
		return err
	}
	return nil
}

func (a *Application) pause() error {
	player, err := a.selectedPlayer()
	if err != nil {
		return err
	}
	_, err = player.Command("pause")
	if err != nil {
		return err
	}
	return nil
}
