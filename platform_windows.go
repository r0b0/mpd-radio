package main

import (
	"errors"
	"golang.org/x/sys/windows/registry"
	"syscall"
)

const REG_PATH = "SOFTWARE\\MpdRadio"
const REG_VALUE = "config"

func loadConfig() (string, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		REG_PATH,
		registry.ALL_ACCESS)
	if err != nil {
		return "", err
	}
	defer k.Close()
	data, _, err := k.GetStringValue(REG_VALUE)
	if err != nil {
		if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			// OK
			return "{}", nil
		}
		return "", err
	}
	return data, nil
}

func saveConfig(data string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		REG_PATH,
		registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()
	err = k.SetStringValue(REG_VALUE, data)
	if err != nil {
		return err
	}
	return nil
}
