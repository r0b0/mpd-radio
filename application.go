package main

import (
    "encoding/json"
    "fmt"
)

type Application struct {
    ServerList []*MpdClient
    UrlList    []Radio
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
