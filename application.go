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

const AppVersion = "24.10"

type Radio struct {
	Name string
	Url  string
}

type Context struct {
	PlayerList    []*MpdClient
	RadioList     []Radio
	Status        string
	IsPlaying     bool
	Volume		  int
	statusUpdated time.Time
	template      *template.Template
	ctx           context.Context
	AppVersion    string
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
}

func (c *Context) RemoveRadio(name string) error {
	for i, r := range c.RadioList {
		if r.Url == name {
			c.RadioList = slices.Delete(c.RadioList, i, i+1)
			err := c.Store()
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("radio not found")
}

func (c *Context) RemovePlayer(address string) error {
	for i, p := range c.PlayerList {
		if p.Address == address {
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

func (c *Context) FindPlayer(url string) (*MpdClient, error) {
	for _, p := range c.PlayerList {
		if p.Address == url {
			return p, nil
		}
	}
	return nil, fmt.Errorf("player not found")
}

func (c *Context) Play(player *MpdClient, url string) error {
	_, err := player.CommandOrReconnect(c.ctx, "clear")
	if err != nil {
		return err
	}
	addIdData, err := player.CommandOrReconnect(c.ctx, fmt.Sprintf("addid \"%s\" 0", url))
	if err != nil {
		return err
	}
	addIdData.Print()
	id, ok := addIdData.Response["Id"]
	if !ok {
		return fmt.Errorf("failed to get id of added song")
	}
	_, err = player.CommandOrReconnect(c.ctx, fmt.Sprintf("playid %s", id))
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Stop(player *MpdClient) error {
	_, err := player.CommandOrReconnect(c.ctx, "stop")
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Pause(player *MpdClient) error {
	_, err := player.CommandOrReconnect(c.ctx, "pause")
	if err != nil {
		return err
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
	player, err := c.FindPlayer(url)
	if err != nil {
		return err
	}
	c.statusUpdated = time.Now()
	statusData, err := player.CommandOrReconnect(c.ctx, "status")
	if err != nil {
		return err
	}
	statusData.Print()
	status, ok := statusData.Response["state"]
	if !ok {
		return fmt.Errorf("failed to get player status")
	}
	switch status {
	case "play":
		songData, err := player.Command("currentsong")
		if err != nil {
			return err
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
    _, err := player.CommandOrReconnect(c.ctx, fmt.Sprintf("setvol %d", c.Volume))
    return err
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
		slog.Error("failed to unmarshall application context; using default empty context: %v", err)
	}
	c.AppVersion = AppVersion
	c.Status = "" // force reload from the server
	return &c
}
