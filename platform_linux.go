package main

import (
	"os"
	"path"
)

const CONFIG_FILE = "mpd-radio-config.json"

func loadConfig() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path.Join(d, CONFIG_FILE))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func saveConfig(data string) error {
	d, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(d, CONFIG_FILE), []byte(data), 0600)
	return err
}
