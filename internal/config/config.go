package config

import (
	"fmt"
	"os"
	"encoding/json"
	"path/filepath"
)

type Config struct {
	DBURL string `json:"db_url"`
	CurrentUsername string `json:"current_user_name"`
}

func Read() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("error getting user home directory: %v", err)
	}
	path := filepath.Join(homeDir, ".gatorconfig.json")
	c := Config{}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("error opening config file at %v: %v", homeDir, err)
	}
	defer file.Close()
	d := json.NewDecoder(file)
	err = d.Decode(&c)
	if err != nil {
		return Config{}, fmt.Errorf("error unmarshaling json: %v", err)
	}
	return c, nil
}

func (c *Config) SetUser(username string) error {
	c.CurrentUsername = username
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %v", err)
	}
	path := filepath.Join(homeDir, ".gatorconfig.json")
	file, err2 := json.Marshal(c)
	if err2 != nil {
		return fmt.Errorf("error marshaling config struct: %v", err2)
	}
	err3 := os.WriteFile(path, file, 0644)
	if err3 != nil {
		return fmt.Errorf("error writing file: %v", err3)
	}
	return nil
}