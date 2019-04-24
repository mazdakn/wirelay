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
    nodes   map[string]NodeType
}

type NodeType struct {
    Name            string   `json:"name"`
    ControlAddr     string   `json:"control"`
    DataAddr        string   `json:"data"`
    Pubkey          string   `json:"pubkey"`
    Networks        []string `json:"networks"`
}

func (e *Engine) registerHandler(w http.ResponseWriter, r *http.Request) {
    var node    NodeType
    action := "Registered"

    if r.Method == "POST" {
        json.NewDecoder(r.Body).Decode(&node)

        if node.Name == "" || node.ControlAddr == "" || node.DataAddr == "" || len(node.Networks) == 0 { //TODO: what about pubkey?
            return
        }

        if _, found := e.nodes[node.Name]; found {
            action = "Updated"
        }

        e.nodes[node.Name] = node

        for _, net := range node.Networks {
            e.data.rules.CompilePolicy(PolicyEntryFile{DstSubnet: net, Action: "FORWARD", Endpoint: node.DataAddr})
        }
        log.Println(e.nodes)
        log.Println(action, r.RemoteAddr)
    }
}

func (e *Engine) exposeHandler(w http.ResponseWriter, r *http.Request) {
    node := NodeType{}

    if r.Method == "GET" {
        node.Name        = e.conf.content.Name
        node.ControlAddr = e.conf.content.Control
        node.DataAddr    = e.conf.content.Data
        node.Pubkey      = e.conf.content.Pubkey
        node.Networks    = e.conf.content.Networks

        json.NewEncoder(w).Encode(&node)
        log.Println("Exposed to", r.RemoteAddr)
    }
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

    e.nodes = make(map[string]NodeType)

    // Setup and register signal handler
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs)
    go e.signalHandler(sigs)

    http.HandleFunc("/register", e.registerHandler)
    http.HandleFunc("/expose", e.exposeHandler)

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
