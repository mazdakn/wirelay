package main

import (
    "log"
    "sync"
    //"errors"
	"os"
	"os/signal"
	"syscall"
)

type EngineEntry struct {
    policy Policy
    netio  NetIO
}

type Engine struct {
    conf Configuration
    interfaces []EngineEntry
}

func (e *Engine) Forward(entry EngineEntry, waitGroup *sync.WaitGroup ) {
    var pkt Packet
    var action PolicyAction
    var found bool

    defer waitGroup.Done()
    for {
        // TODO: set MTU and read correct number of bytes
        pkt.Data = make([]byte, 2000)

        if err := entry.netio.Receive(&pkt); err != nil {
            log.Print (err)
            continue
        }

        if !pkt.IsIPv4() {
            continue
        }

        // TODO: add counters

        if action, found = entry.policy.Lookup (&pkt); !found {
            //TODO: counter for dropped packets
            continue
        }

        if e.interfaces[action.egress].netio == nil {
            //TODO: log
            continue
        }

        pkt.Endpoint = action.endpoint
        if err := e.interfaces[action.egress].netio.Send(&pkt); err != nil {
            log.Println (err)
        }
    }
}

func (e *Engine) Start() {
    var waitGroup sync.WaitGroup

	log.Println ("Starting wirelay core")

    for index, entry := range e.interfaces {
        if entry.netio != nil{
            log.Println("Starting interface", index, entry)
            go e.Forward(entry, &waitGroup)
            waitGroup.Add(1)
        }
    }

	waitGroup.Wait()
	log.Println ("Shuting down")
}

func (e *Engine) Init () (error) {
    var entry EngineEntry

    e.interfaces = make([]EngineEntry, 10)

    for _, netio := range e.conf.content {

        if netio.Type == "NULL" {
            entry.netio = &Null{}
        }

        if netio.Type == "LOCAL" {
            entry.netio = &TunTap{Name: netio.Name}
        }

        if netio.Type == "TUNNEL" {
            entry.netio = &Tunnel{LocalSocket: netio.Address}
        }

        if err := entry.netio.Init(); err != nil {
            return err
        }

        entry.policy = Policy{}
        entry.policy.Init()
        if err := entry.policy.CompilePolicies(netio.Policies); err != nil {
            return err
        }

        log.Println("Interface:", netio.Name)
        entry.policy.Dump()

        e.interfaces[netio.ID] = entry
    }

	e.setupSignal()
	log.Println("Singnal initiated")

    return nil
}


// signal handler for Interrupt, Terminate, and SIGHUP
func (e *Engine) signalHandler(signal chan os.Signal) {
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
func (e *Engine) setupSignal() {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs)
    go e.signalHandler(sigs)
}
