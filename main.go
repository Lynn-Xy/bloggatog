package main

import {
	cfg "bloggatog/internal/config/.gatorconfig"
	"fmt"
	"os"
}

func main() {
	// create a buffered io reader
	cont, err := cfg.Read()
	if err {
		fmt.Printf("error reading config: %v", err)
	}
	s := &cfg.State{config: cont}
	commands := cfg.Commands{}
	commands.Register("login", login(s, os.Args[1]))
	continue
}
