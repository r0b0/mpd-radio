package main

import (
	"context"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}

type Application struct {
	PlayerList     []*MpdClient
	RadioList      []Radio
	fyneApp        fyne.App
	fyneParent     fyne.Window
	playerDropdown *widget.Select
	radioDropdown  *widget.Select
	statusLabel    *widget.Label
	statusIcon     *widget.Activity
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

func (a *Application) selectedPlayer() (int, *MpdClient, error) {
	for i, s := range a.PlayerList {
		if s.Address == a.playerDropdown.Selected {
			return i, s, nil
		}
	}
	return -1, nil, &NotFoundError{}
}

func (a *Application) showRadioList() []string {
	ret := []string{"Add New..."}
	for _, s := range a.RadioList {
		ret = append(ret, s.Name)
	}
	return ret
}

func (a *Application) selectedRadio() (int, *Radio, error) {
	for i, r := range a.RadioList {
		if r.Name == a.radioDropdown.Selected {
			return i, &r, nil
		}
	}
	return -1, nil, &NotFoundError{}
}

func (a *Application) getPlayerStatus() (string, bool, error) {
	_, player, err := a.selectedPlayer()
	if err != nil {
		return "", false, err
	}
	data, err := player.Command("status")
	if err != nil {
		return "", false, err
	}
	data.Print()
	status, ok := data.Response["state"]
	if !ok {
		return "", false, fmt.Errorf("failed to get player status")
	}
	switch status {
	case "play":
		songData, err := player.Command("currentsong")
		if err != nil {
			return "", false, err
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
		return fmt.Sprintf("Playing: %s", playing), true, nil
	case "stop":
		return "Stopped", false, nil
	case "pause":
		return "Paused", false, nil
	}
	return "", false, nil

}

func (a *Application) play() error {
	_, player, err := a.selectedPlayer()
	if err != nil {
		return err
	}
	_, radio, err := a.selectedRadio()
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
	_, player, err := a.selectedPlayer()
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
	_, player, err := a.selectedPlayer()
	if err != nil {
		return err
	}
	_, err = player.Command("pause")
	if err != nil {
		return err
	}
	return nil
}
