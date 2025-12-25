package main

import {
	"bloggatog/internal/config/gatorconfig"
	"fmt"
}

func main() {
	cont, err := gatorconfig.Read()
	if err {
		fmt.Printf("error reading config: %v", err)
	}
	cont.SetUser("lynn")
	cont2, err2 := gatorconfig.Read()
	if err2 {
		fmt.Printf("error reading config: %v", err2)
	}
}
