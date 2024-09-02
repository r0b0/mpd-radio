package main

import (
    "encoding/json"
    "fmt"
)

type Application struct {
    PlayerList []*MpdClient
    RadioList  []Radio
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
