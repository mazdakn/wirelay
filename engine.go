package main

import (
    "sync"
	"os"
	"os/signal"
	"syscall"
	"net/http"
	"log"
	"encoding/json"
)

type Engine struct {
    conf    Configuration
    data    DataPlane
    nodes   map[string]NodesType
}

type NodesType struct {
    ControlAddr     string   `json:"control"`
    DataAddr        string   `json:"data"`
    Pubkey          string   `json:"pubkey"`
    Networks        []string `json:"networks"`
}

func (e *Engine) registerNode(w http.ResponseWriter, r *http.Request) {
   var node NodesType

   json.NewDecoder(r.Body).Decode(&node)
   log.Println(node)
   log.Println("Registered =>", r.Host)
}

func (e *Engine) expose(w http.ResponseWriter, r *http.Request) {
    node := NodesType{}

    //log.Println(e.conf.content)
    node.ControlAddr = e.conf.content.Control
    node.DataAddr    = e.conf.content.Data
    node.Pubkey      = e.conf.content.Pubkey
     node.Networks    = []string{} //TODO: Fix it

    log.Println(node)
    json.NewEncoder(w).Encode(&node)
    log.Println("About Me =>", r.Host)
}

/* Initilizing the Wirelay Engine
   1 - Create and initialize the slice of interfaces from the Configuration
   2 - Create Policy object for each interface
   3 - Setup the signal handler
*/
func (e *Engine) Init() {
    var err   error

    err = e.conf.Init()
    Fatal(err)

    e.data.conf = &(e.conf)
    err = e.data.Init()
    Fatal(err)

    // Setup and register signal handler
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs)
    go e.signalHandler(sigs)

    http.HandleFunc("/register", e.registerNode)
    http.HandleFunc("/about", e.expose)

    go http.ListenAndServe(e.conf.content.Control, nil)
}

// signal handler for Interrupt, Terminate, and SIGHUP
func (e *Engine) signalHandler(signal chan os.Signal) {
    for {
        sig := <-signal
        switch sig {
        case syscall.SIGUSR1:
            e.data.PrintCounters()
        case syscall.SIGUSR2:
            e.data.rules.DumpPolicies()
        case os.Interrupt, syscall.SIGTERM:
            Print("Shutting down")
            os.Exit(0)
            // TODO: use channel to shutdown gracefully
        case syscall.SIGHUP:
            // TODO: reload configuration from config file and refresh all connections  
            Print("Reloading configuration")
        }
    }
}

func (e *Engine) Start() {
    var waitGroup sync.WaitGroup

	Print("Starting wirelay dataplane")
    dataGo := e.data.Start(&waitGroup)
    waitGroup.Add(dataGo)

	waitGroup.Wait()
	Print("Shuting down")
}


