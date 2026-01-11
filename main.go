package main

import (
	cfg "github.com/Lynn-Xy/bloggatog/internal/config"
	"fmt"
	"os"
	"errors"
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/Lynn-Xy/bloggatog/internal/database"
	"log"
	"time"
)

type state struct {
	cfg *cfg.Config
	db *database.Queries
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
	_, err := s.db.GetUserByName(context.Background(), cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error retrieving username from database: %v", err)
	}
	err = s.cfg.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting username: %v", err)
	}
	fmt.Printf("current user: %v has been set\n", cmd.Arguments[0])
	return nil
}

func HandlerRegister(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("error registering user: no username provided")
	}
	_, err := s.db.GetUserByName(context.Background(), cmd.Arguments[0])
	if err == nil {
		return fmt.Errorf("error registering new user: %v - user already exists in database", cmd.Arguments[0])
	}
	params := database.CreateUserParams{uuid.New(), time.Now(), time.Now(), cmd.Arguments[0]}
	_, err = s.db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error creating new user: %v", err)
	}
	err = s.cfg.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting current user: %v", err)
	}
	fmt.Printf("current user: %v has been registered\n", cmd.Arguments[0])
	return nil
}

func HandlerUsers(s *state, cmd command) error {
	users, err := s.db.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error retrieving users from database: %v", err)
	}
	for _, user := range users {
		if user.Name == s.cfg.CurrentUsername {
			fmt.Printf("* %v (current)\n", user.Name)
		} else {
			fmt.Printf("* %v\n", user.Name)
		}
	}
	return nil
}

func HandlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error deleting users from database: %v", err)
	}
	fmt.Print("all users deleted from database")
	return nil
}

func main() {
	config, err := cfg.Read()
	if err != nil {
		fmt.Printf("error reading config: %v", err)
		os.Exit(1)
	}
	db, err := sql.Open("postgres", config.DBURL)
	if err != nil {
		log.Printf("Error opening database connection: %v", err)
	}
	dbQ := database.New(db)
	s := &state{&config, dbQ}
	commands := commands{
		list: make(map[string]func(*state, command) error),
	}

	commands.Register("login", HandlerLogin)
	commands.Register("register", HandlerRegister)
	commands.Register("reset", HandlerReset)
	commands.Register("users", HandlerUsers)

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
