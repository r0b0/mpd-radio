package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"slices"
	"strconv"
	"time"
)

const AppVersion = "26.03"

type Radio struct {
	Name string
	Url  string
}

type Context struct {
	PlayerList     []*MpdClient
	SelectedPlayer int
	RadioList      []Radio
	SelectedRadio  int
	Status         string
	IsPlaying      bool
	Volume         int
	statusUpdated  time.Time
	template       *template.Template
	ctx            context.Context
	AppVersion     string
}

func (c *Context) ConnectPlayer(p *MpdClient) {
	if p.logger == nil {
		p.logger = slog.With("player address", p.Address)
	}
	err := p.Connect(c.ctx)
	if err != nil {
		slog.Error("Failed to connect player %s: %s", p.Address, err)
		return
	}
	if c.Status == "" {
		_ = c.UpdateStatus(p.Address)
	}
	_ = c.FindPlayer(p.Address) // just to mark it selected
}

func (c *Context) RemoveRadio(name string) error {
	for i, r := range c.RadioList {
		if r.Url == name {
			c.RadioList = slices.Delete(c.RadioList, i, i+1)
			err := c.Store()
			if err != nil {
				return err
			}
			c.SelectedRadio = 0
			return nil
		}
	}
	return fmt.Errorf("radio not found")
}

func (c *Context) RemovePlayer(address string) error {
	for i, p := range c.PlayerList {
		if p.Address == address {
			c.SelectedPlayer = 0
			p.Close()
			c.PlayerList = slices.Delete(c.PlayerList, i, i+1)
			err := c.Store()
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("player not found")
}

func (c *Context) FindPlayer(url string) *MpdClient {
	for i, p := range c.PlayerList {
		if p.Address == url {
			c.SelectedPlayer = i
			return p
		}
	}
	return nil
}

func (c *Context) FindRadio(url string) *Radio {
	for i, r := range c.RadioList {
		if r.Url == url {
			c.SelectedRadio = i
			return &r
		}
	}
	return nil
}

func (c *Context) Play(player *MpdClient, url string) error {
	_ = c.FindPlayer(player.Address) // just to mark it selected
	_ = c.FindRadio(url)             // just to mark it selected
	r := player.Command(c.ctx, "clear")
	if r.Error != nil {
		return r.Error
	}
	addIdData := player.Command(c.ctx, fmt.Sprintf("addid \"%s\" 0", url))
	if addIdData.Error != nil {
		return addIdData.Error
	}
	addIdData.Print()
	id, ok := addIdData.Response["Id"]
	if !ok {
		return fmt.Errorf("failed to get id of added song")
	}
	resp := player.Command(c.ctx, fmt.Sprintf("playid %s", id))
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (c *Context) Stop(player *MpdClient) error {
	_ = c.FindPlayer(player.Address) // just to mark it selected
	resp := player.Command(c.ctx, "stop")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (c *Context) Pause(player *MpdClient) error {
	_ = c.FindPlayer(player.Address) // just to mark it selected
	resp := player.Command(c.ctx, "pause")
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}

func (c *Context) Store() error {
	j, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return saveConfig(j)
}

func (c *Context) UpdateStatus(url string) error {
	if time.Now().Before(c.statusUpdated.Add(10 * time.Second)) {
		slog.Debug("status is still ok, no need to fetch")
		return nil
	}
	player := c.FindPlayer(url)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	c.statusUpdated = time.Now()
	statusData := player.Command(c.ctx, "status")
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
		songData := player.Command(c.ctx, "currentsong")
		if songData.Error != nil {
			return songData.Error
		}
		songData.Print()
		tags := []string{"Title", "Name", "file"}
		for _, tag := range tags {
			name, ok := songData.Response[tag]
			if ok {
				c.Status = name
				break
			}
		}
		c.IsPlaying = true
	case "stop":
		c.Status = "Stopped"
		c.IsPlaying = false
	case "pause":
		c.Status = "Paused"
		c.IsPlaying = false
	}

	volume, ok := statusData.Response["volume"]
	if !ok {
		slog.Warn("Failed to fetch volume from status response")
		volume = "0"
	}
	var err error
	c.Volume, err = strconv.Atoi(volume)
	if err != nil {
		slog.Warn("Failed to parse volume from status response")
		c.Volume = 0
	}
	if c.Volume < 0 || c.Volume > 100 {
		slog.Warn("Invalid volume value from status respose")
		c.Volume = 0
	}

	return nil
}

func (c *Context) UpdateVolume(player *MpdClient, change int) error {
	c.Volume += change

	if c.Volume < 0 {
		c.Volume = 0
	}
	if c.Volume > 100 {
		c.Volume = 100
	}
	resp := player.Command(c.ctx, fmt.Sprintf("setvol %d", c.Volume))
	return resp.Error
}

func Load() *Context {
	j, err := loadConfig()
	c := Context{}
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
