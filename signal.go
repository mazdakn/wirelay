package main

import (
	"os"
    "os/signal"
	"syscall"
	"log"
)

// signal handler for Interrupt, Terminate, and SIGHUP
func signalHandler(signal chan os.Signal) {
    for {
        sig := <-signal
        switch sig {
        case os.Interrupt, syscall.SIGTERM:
            log.Println ("Shutting down")
            // TODO: use channel to shutdown gracefully
            os.Exit (1)
        case syscall.SIGHUP:
            // TODO: reload configuration from config file and refresh all connections  
            log.Println ("Reloading configuration")
        }
    }
}

// setup and register signal handler
func setupSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs)
	go signalHandler(sigs)
}
