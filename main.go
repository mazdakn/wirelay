package main

import (
    "os"
	"flag"
    "fmt"
)

type Arguments struct {
    config string
}

// parse CLI arguments
func parseArguments() (Arguments) {
	var args Arguments

	version := flag.Bool("v", false, "Print version")
    config := flag.String ("c", "config.json", "Configuration file")
    flag.Parse()

    if (*version) {
        fmt.Println ("Wirelay - Version 0.1")
        os.Exit(0)
    }

    args.config = *config

    return args
}

func main() {

    args := parseArguments()

    setupSignal() // setup signal handlers

    var engine Engine
    engine.Configfile = args.config

    if err := engine.Init(); err != nil {
        fmt.Println (err)
        os.Exit(1)
    }

    engine.Start()
}
