package main

import (
    "log"
    "sync"
	"os"
	"os/signal"
	"syscall"
	"net"
	"errors"
	"strconv"
)

type NetIO interface {
    Init() (error)
    Close() (error)
    Receive(*Packet) (error)
    Send(*Packet) (error)
}

const (
    NETIO_LOCAL    uint8 = 0
    NETIO_FORWARD  uint8 = 1
    NETIO_DROP     uint8 = 2
    NETIO_MAX      uint8 = 3
)

type Counters struct {
    Received    uint32
    Sent        uint32
    Dropped     uint32
    ErrReceive  uint32
    ErrSend     uint32
    UnSupported uint32
}

type EngineEntry struct {
    netio       NetIO
    rules       Policy
    counters    Counters
}

type Engine struct {
    conf Configuration
    nodes [NETIO_MAX]EngineEntry
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

    // Create local TUN interface
    e.nodes[NETIO_LOCAL].netio = &TunTap{Name: e.conf.content.Name}
    err = e.nodes[NETIO_LOCAL].netio.Init()
    Fatal(err)
    err = e.nodes[NETIO_LOCAL].rules.CompilePolicies(e.conf.content.Policies)
    Fatal(err)

    e.nodes[NETIO_FORWARD].netio = &Tunnel{LocalSocket: e.conf.content.Address}
    err = e.nodes[NETIO_FORWARD].netio.Init()
    Fatal(err)
    err = e.nodes[NETIO_FORWARD].rules.CompilePolicies(e.conf.content.Policies)
    Fatal(err)

    e.nodes[NETIO_DROP].netio = &Drop{}
    err = e.nodes[NETIO_DROP].netio.Init()
    Fatal(err)

    // Setup and register signal handler
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs)
    go e.signalHandler(sigs)
}


// signal handler for Interrupt, Terminate, and SIGHUP
func (e *Engine) signalHandler(signal chan os.Signal) {
    for {
        sig := <-signal
        switch sig {
        case syscall.SIGUSR1:
            e.PrintCounters()
        case syscall.SIGUSR2:
            e.DumpPolicies()
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

	Print("Starting wirelay core")

    go e.Forward(&e.nodes[NETIO_LOCAL], &waitGroup)
    go e.Forward(&e.nodes[NETIO_FORWARD], &waitGroup)
    waitGroup.Add(2)

	waitGroup.Wait()
	Print("Shuting down")
}


func (e *Engine) Forward(dev *EngineEntry, waitGroup *sync.WaitGroup) {
    var pkt Packet
    var action PolicyAction
    var found bool

   // TODO: set MTU and read correct number of bytes
    pkt.Data = make([]byte, 2000)

    defer waitGroup.Done()
    for {
        if err := dev.netio.Receive(&pkt); err != nil {
            dev.counters.ErrReceive++
            continue
        }

        dev.counters.Received++

        if !pkt.IsIPv4() {
            dev.counters.UnSupported++
            continue
        }

        if action, found = e.Lookup(&pkt); !found {
            dev.counters.Dropped++
            continue
        }

        pkt.Endpoint = action.endpoint
        if err := e.nodes[action.egress].netio.Send(&pkt); err != nil {
            dev.counters.ErrSend++
            continue
        }

        dev.counters.Sent++
    }
}


func (e *Engine) PrintCounters() {

    links := []EngineEntry{e.local, e.tunnel}
    names := []string{"Local", "Tunnel"}
    Print("Engine counters:")

    for index, entry := range links {
        log.Println(names[index] + ":")
        log.Println("\tReceived:\t", entry.counters.Received)
        log.Println("\tSent:\t\t", entry.counters.Sent)
        log.Println("\tDropped:\t", entry.counters.Dropped)
        log.Println("\tUnsupported:\t", entry.counters.UnSupported)
        log.Println("\tError Receive:\t", entry.counters.ErrReceive)
        log.Println("\tError Send:\t", entry.counters.ErrSend)
    }
}
