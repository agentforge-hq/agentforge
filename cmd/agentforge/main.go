package main

import (
	"os"

	forge "github.com/agentforge-hq/agentforge"
)

func main() { os.Exit(forge.Run(os.Args[1:], os.Stdout, os.Stderr)) }
