package main

import (
	cfg "github.com/bloggatog/internal/config"
	"fmt"
	"os"
	"errors"
	_ "github.com/lib/pq"
)

type state struct {
	config *cfg.Config
}

type command struct {
	Name string
	Arguments []string
}

type commands struct {
	list map[string]func(*state, command) error
}

func (c *commands) Run(s *state, cmd command) error {
	handler, ok := c.list[cmd.Name]
	if !ok {
		return fmt.Errorf("error running cmd: %v", cmd.Name)
	}
	err := handler(s, cmd)
	if err != nil {
		return fmt.Errorf("error executing command %v: %v", cmd.Name, err)
	}
	return nil
}

func (c *commands) Register(name string, f func(*state, command) error) {
	c.list[name] = f
}

func HandlerLogin(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("error logging in: current username not found")
}
	err := s.config.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting username: %v", err)
	}
	fmt.Printf("current user: %v has been set\n", cmd.Arguments[0])
	return nil
}

func main() {
	config, err := cfg.Read()
	if err != nil {
		fmt.Printf("error reading config: %v", err)
		os.Exit(1)

	}

	s := &state{config: &config}

	commands := commands{
		list: make(map[string]func(*state, command) error),
	}

	commands.Register("login", HandlerLogin)

	args := os.Args
	if len(args) < 2 {
		fmt.Println("not enough arguments provided")
		os.Exit(1)
	}

	cmd := command{
		Name:      args[1],
		Arguments: args[2:],
	}

	if err := commands.Run(s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
