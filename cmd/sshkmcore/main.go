package main

import (
	"log"
	"os"

	"github.com/RTHeLL/ssh-keys-manager/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
