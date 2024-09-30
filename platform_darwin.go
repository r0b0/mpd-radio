package main

import (
	"os"
	"path"
)

const CONFIG_FILE = "mpd-radio-config.json"

func loadConfig() ([]byte, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return []byte{}, err
	}
	data, err := os.ReadFile(path.Join(d, CONFIG_FILE))
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func saveConfig(data []byte) error {
	d, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(d, CONFIG_FILE), data, 0600)
	return err
}
