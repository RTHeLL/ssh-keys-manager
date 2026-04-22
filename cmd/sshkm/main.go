package main

import (
	"log"
	"os"

	"github.com/RTHeLL/ssh-keys-manager/internal/cli"
)

func main() {
	root := cli.NewRootCommand()
	if err := root.Execute(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
