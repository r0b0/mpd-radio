package main

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
)

type Application struct {
	PlayerList []*MpdClient
	RadioList  []Radio
	fyneApp    fyne.App
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

func (a *Application) selectedPlayer(name string) (*MpdClient, error) {
	for _, s := range a.PlayerList {
		if s.Address == name {
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

func (a *Application) selectedRadio(name string) (*Radio, error) {
	for _, r := range a.RadioList {
		if r.Name == name {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("radio not found")
}
