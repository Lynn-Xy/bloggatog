package config

import {
	"fmt"
	"os"
	"encoding/json"
}

type Config struct {
	db_url string `json:"db_url"`
	current_user_name string `json:"current_user_name"`
}


func Read() *Config, error {
	homeDir := os.UserHomeDir()
	c := &Config{}
	file, err := os.Open(homeDir) {
		if err {
			return fmt.Errorf("error opening config file at %v: %v", hokeDir, err)
		}
	}
	cont, err := file.Read(homeDir + "/gatorconfig.json") {
		if err {
			return fmt.Errorf("error reading confif file at %v: %v", homeDir, err)
		}
	}
	if err := json.Unmarshal(cont, &c); err == true {
		return fmt.Errorf("error unmarshaling json: %v", err)
	}
	return c
}

func (c Config) SetUser(username string) {
	c.current_user_name = username
	homeDir := os.UserHomeDir()
	file, err := json.Marshal(c)
	if err {
		fmt.Printf("error marshaling config struct: %v", err)
	}
	err := os.WriteFile("gatorconfig.json", file)
	if err {
		fmt.Printf("error writing file: %v", err)
	}
}
