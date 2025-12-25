package config

import {
	"fmt"
	"os"
	"encoding/json"
	"errors"
}

type Config struct {
	db_url string `json:"db_url"`
	current_user_name string `json:"current_user_name"`
}

type State struct {
	config *Config
}

type Command struct {
	name string
	arguments []string
}

type Commands struct {
	list map[string]func(*state, command) error
}

func (c *commands) Run(s *state, cmd command) error {
	err := c.name.cmd(s, cmd)
	if err {
		return fmt.Errorf("error running cmd: %v: %v", c.name, err)
	}
}

func (c *commands Register(name string, f func(*state, command) error {
	c.list[name] = f
	return nil
}

func Read() *Config, error {
	homeDir := os.UserHomeDir()
	c := &Config{}
	file, err := os.Open(homeDir) {
		if err {
			return fmt.Errorf("error opening config file at %v: %v", hokeDir, err)
		}
	}
	cont, err := file.Read(homeDir + "/.gatorconfig.json") {
		if err {
			return fmt.Errorf("error reading confif file at %v: %v", homeDir, err)
		}
	}
	if err := json.Unmarshal(cont, &c); err == true {
		return fmt.Errorf("error unmarshaling json: %v", err)
	}
	return c
}

func (c Config) SetUser(username string) error {
	c.current_user_name = username
	homeDir := os.UserHomeDir()
	file, err := json.Marshal(c)
	if err {
		return fmt.Errorf("error marshaling config struct: %v", err)
	}
	err2 := os.WriteFile("gatorconfig.json", file)
	if err2 {
		return fmt.Errorf("error writing file: %v", err)
	}
}

func HandlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) < 1 {
		return errors.New("error logging in: current username not found")
}
	err := s.config.SetUser(cmd.arguments[0])
	if err {
		return fmt.Errorf("error setting username: %v", err)
	}
	fmt.Printf("current user: %v has been set", cmd.argument[0])
	return nil
}
