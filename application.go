package main

import (
	"context"
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"log/slog"
	"slices"
	"strconv"
	textTemplate "text/template"
	"time"
)

const AppVersion = "26.05"

type Radio struct {
	Name string
	Url  string
}

type Application struct {
	PlayerList     []*MpdClient
	SelectedPlayer int
	RadioList      []Radio
	SelectedRadio  int
	Status         string
	IsPlaying      bool
	Volume         int
	statusUpdated  time.Time
	htmlTemplate   *htmlTemplate.Template
	textTemplate   *textTemplate.Template
	ctx            context.Context
	AppVersion     string
}

func (a *Application) ConnectPlayer(p *MpdClient) {
	if p.logger == nil {
		p.logger = slog.With("player address", p.Address)
	}
	err := p.Connect(a.ctx)
	if err != nil {
		slog.Error("Failed to connect player %s: %s", p.Address, err)
		return
	}
	if a.Status == "" {
		_ = a.UpdateStatus(p.Address)
	}
	_ = a.FindPlayer(p.Address) // just to mark it selected
}

func (a *Application) RemoveRadio(name string) error {
	found := false
	a.RadioList = slices.DeleteFunc(a.RadioList, func(r Radio) bool {
		found = true
		return r.Url == name
	})
	if found {
		err := a.Store()
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("radio not found")
}

func (a *Application) RemovePlayer(address string) error {
	found := false
	a.PlayerList = slices.DeleteFunc(a.PlayerList, func(p *MpdClient) bool {
		found = true
		return p.Address == address
	})
	if found {
		err := a.Store()
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("player not found")
}

func (a *Application) FindPlayer(url string) *MpdClient {
	for i, p := range a.PlayerList {
		if p.Address == url {
			a.SelectedPlayer = i
			return p
		}
	}
	if url == "" && len(a.PlayerList) > 0 {
		return a.PlayerList[0]
	}
	return nil
}

func (a *Application) FindRadio(url string) *Radio {
	for i, r := range a.RadioList {
		if r.Url == url {
			a.SelectedRadio = i
			return &r
		}
	}
	return nil
}

func (a *Application) Play(player *MpdClient, url string) error {
	_ = a.FindPlayer(player.Address) // just to mark it selected
	_ = a.FindRadio(url)             // just to mark it selected
	r := player.Command(a.ctx, "clear")
	if r.Error != nil {
		return r.Error
	}
	addIdData := player.Command(a.ctx, fmt.Sprintf("addid \"%s\" 0", url))
	if addIdData.Error != nil {
		return addIdData.Error
	}
	addIdData.Print()
	id, ok := addIdData.Response["Id"]
	if !ok {
		return fmt.Errorf("failed to get id of added song")
	}
	resp := player.Command(a.ctx, fmt.Sprintf("playid %s", id))
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (a *Application) Stop(player *MpdClient) error {
	_ = a.FindPlayer(player.Address) // just to mark it selected
	resp := player.Command(a.ctx, "stop")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (a *Application) Pause(player *MpdClient) error {
	_ = a.FindPlayer(player.Address) // just to mark it selected
	resp := player.Command(a.ctx, "pause")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (a *Application) Store() error {
	j, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return saveConfig(j)
}

func (a *Application) UpdateStatus(url string) error {
	if time.Now().Before(a.statusUpdated.Add(10 * time.Second)) {
		slog.Debug("status is still ok, no need to fetch")
		return nil
	}
	player := a.FindPlayer(url)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	a.statusUpdated = time.Now()
	statusData := player.Command(a.ctx, "status")
	if statusData.Error != nil {
		return statusData.Error
	}
	statusData.Print()
	status, ok := statusData.Response["state"]
	if !ok {
		return fmt.Errorf("failed to get player status")
	}
	switch status {
	case "play":
		songData := player.Command(a.ctx, "currentsong")
		if songData.Error != nil {
			return songData.Error
		}
		songData.Print()
		tags := []string{"Title", "Name", "file"}
		for _, tag := range tags {
			name, ok := songData.Response[tag]
			if ok {
				a.Status = name
				break
			}
		}
		a.IsPlaying = true
	case "stop":
		a.Status = "Stopped"
		a.IsPlaying = false
	case "pause":
		a.Status = "Paused"
		a.IsPlaying = false
	}

	volume, ok := statusData.Response["volume"]
	if !ok {
		slog.Warn("Failed to fetch volume from status response")
		volume = "0"
	}
	var err error
	a.Volume, err = strconv.Atoi(volume)
	if err != nil {
		slog.Warn("Failed to parse volume from status response")
		a.Volume = 0
	}
	if a.Volume < 0 || a.Volume > 100 {
		slog.Warn("Invalid volume value from status respose")
		a.Volume = 0
	}

	return nil
}

func (a *Application) UpdateVolume(player *MpdClient, change int) error {
	a.Volume += change

	if a.Volume < 0 {
		a.Volume = 0
	}
	if a.Volume > 100 {
		a.Volume = 100
	}
	resp := player.Command(a.ctx, fmt.Sprintf("setvol %d", a.Volume))
	return resp.Error
}

func Load() *Application {
	j, err := loadConfig()
	c := Application{}
	c.AppVersion = AppVersion
	if err != nil {
		return &c
	}
	err = json.Unmarshal(j, &c)
	if err != nil {
		slog.Error("failed to unmarshall application context; using default empty context", slog.Any("error", err))
	}
	c.AppVersion = AppVersion
	c.Status = "" // force reload from the server
	return &c
}
