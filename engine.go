package main

import (
    "log"
    "sync"
	"os"
	"os/signal"
	"syscall"
)

type EngineEntry struct {
    policy      Policy
    netio       NetIO
    counters    Counters
}

type Counters struct {
    Received    uint32
    Sent        uint32
    Dropped     uint32
    ErrReceive  uint32
    ErrSend     uint32
    UnSupported uint32
}


type Engine struct {
    conf Configuration
    interfaces []EngineEntry
}

/* Initilizing the Wirelay Engine
   1 - Create and initialize the slice of interfaces from the Configuration
   2 - Create Policy object for each interface
   3 - Setup the signal handler
*/
func (e *Engine) Init () {
    var entry EngineEntry
    var err   error

    err = e.conf.Init()
    Fatal(err)

    e.interfaces = make([]EngineEntry, 0)

    for _, netio := range e.conf.content {

        switch netio.Type {
        case "LOCAL":
            entry.netio = &TunTap{Name: netio.Name}
        case "TUNNEL":
            entry.netio = &Tunnel{LocalSocket: netio.Address}
        default:
            log.Println ("Invalid NetIO type")
        }

        err = entry.netio.Init()
        Fatal(err)

        entry.policy = Policy{}
        entry.policy.Init()
        err = entry.policy.CompilePolicies(netio.Policies)
        Fatal(err)

        Print("Interface:" + netio.Name)
        entry.policy.Dump()

        e.interfaces = append(e.interfaces, entry)
    }

    // Setup and register signal handler
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs)
    go e.signalHandler(sigs)
	Print("Singnal initiated")
}


// signal handler for Interrupt, Terminate, and SIGHUP
func (e *Engine) signalHandler(signal chan os.Signal) {
    for {
        sig := <-signal
        switch sig {
        case syscall.SIGUSR1:
            e.PrintCounters()
        case os.Interrupt, syscall.SIGTERM:
            Exit("Shutting down")
            // TODO: use channel to shutdown gracefully
        case syscall.SIGHUP:
            // TODO: reload configuration from config file and refresh all connections  
            Print("Reloading configuration")
        }
    }
}


func (e *Engine) Start() {
    var waitGroup sync.WaitGroup

	Print("Starting wirelay core")

    for _, entry := range e.interfaces {
        if entry.netio != nil{
            go e.Forward(entry, &waitGroup)
            waitGroup.Add(1)
        }
    }

	waitGroup.Wait()
	Print("Shuting down")
}


func (e *Engine) Forward(entry EngineEntry, waitGroup *sync.WaitGroup) {
    var pkt Packet
    var action PolicyAction
    var found bool

    defer waitGroup.Done()
    for {
        // TODO: set MTU and read correct number of bytes
        pkt.Data = make([]byte, 2000)

        if err := entry.netio.Receive(&pkt); err != nil {
            entry.counters.ErrReceive++
            continue
        }

        entry.counters.Received++

        if !pkt.IsIPv4() {
            entry.counters.UnSupported++
            continue
        }

        if action, found = entry.policy.Lookup(&pkt); !found {
            entry.counters.Dropped++
            continue
        }

        if e.interfaces[action.egress].netio == nil {
            entry.counters.Dropped++
            continue
        }

        pkt.Endpoint = action.endpoint
        if err := e.interfaces[action.egress].netio.Send(&pkt); err != nil {
            entry.counters.ErrSend++
            continue
        }

        entry.counters.Sent++

    }
}


func (e *Engine) PrintCounters() {
    for index, entry := range e.interfaces {
        log.Println("Interface ", index)
        log.Println("\tReceived:\t", entry.counters.Received)
        log.Println("\tSent:\t\t", entry.counters.Sent)
        log.Println("\tDropped:\t", entry.counters.Dropped)
        log.Println("\tUnsupported:\t", entry.counters.UnSupported)
        log.Println("\tError Receive:\t", entry.counters.ErrReceive)
        log.Println("\tError Send:\t", entry.counters.ErrSend)
    }
}
