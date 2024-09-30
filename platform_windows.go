package main

import (
	"errors"
	"golang.org/x/sys/windows/registry"
	"syscall"
)

const REG_PATH = "SOFTWARE\\MpdRadio"
const REG_VALUE = "config"

func loadConfig() ([]byte, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		REG_PATH,
		registry.ALL_ACCESS)
	if err != nil {
		return []byte{}, err
	}
	defer k.Close()
	data, _, err := k.GetStringValue(REG_VALUE)
	if err != nil {
		if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			// OK
			return []byte{}, nil
		}
		return []byte{}, err
	}
	return []byte(data), nil
}

func saveConfig(data []byte) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		REG_PATH,
		registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()
	err = k.SetStringValue(REG_VALUE, string(data))
	if err != nil {
		return err
	}
	return nil
}
