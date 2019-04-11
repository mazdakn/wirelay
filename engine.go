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

var (
    ErrPolicyEmptyMatch   = errors.New("Match criteria is not defined")
    ErrPolicyEmptyAction  = errors.New("Action is node defined")
    ErrPolicyIndexInvalid = errors.New("Index out of bound")
)

type NetIO interface {
    Init() (error)
    Close() (error)
    Receive(*Packet) (error)
    Send(*Packet) (error)
}

type PolicyMatch struct {
    dstSubnet   *net.IPNet
    srcSubnet   *net.IPNet
}

type PolicyAction struct {
    egress      NetIO
    endpoint    *net.UDPAddr
}

type PolicyEntry struct {
    Match  PolicyMatch
    Action PolicyAction
    TimeToLive  int
}

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
    counters    Counters
}

type Engine struct {
    conf Configuration
    policy []PolicyEntry
    links  []EngineEntry
    local  EngineEntry
    tunnel EngineEntry
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
    e.local.netio = &TunTap{Name: e.conf.content.Name}
    err = e.local.netio.Init()
    Fatal(err)

    e.tunnel.netio = &Tunnel{LocalSocket: e.conf.content.Address}
    err = e.tunnel.netio.Init()
    Fatal(err)

    err = e.CompilePolicies(e.conf.content.Policies)
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

    go e.Forward(&e.local, &waitGroup)
    go e.Forward(&e.tunnel, &waitGroup)
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

        if action.egress == nil {
            dev.counters.Dropped++
            continue
        }

        pkt.Endpoint = action.endpoint
        if err := action.egress.Send(&pkt); err != nil {
            dev.counters.ErrSend++
            continue
        }

        dev.counters.Sent++
    }
}


func (e *Engine) CompilePolicies(policies []PolicyEntryFile) (error) {

    tmpPolicy := make([]PolicyEntry, 0)

    for _, pol := range policies {
        var subnet *net.IPNet
        var endpoint *net.UDPAddr
        var err error

        entry := PolicyEntry{}

        if pol.DstSubnet != "" {
            if _, subnet, err = net.ParseCIDR(pol.DstSubnet); err != nil {
                // TODO: log and continue
                return err
            }

            entry.Match.dstSubnet = subnet
        }

        if pol.SrcSubnet != "" {
            if _, subnet, err = net.ParseCIDR(pol.SrcSubnet); err != nil {
                //TODO: log and continue
                return err
            }

            entry.Match.srcSubnet = subnet
        }

        if pol.ttl > 0 {
            entry.TimeToLive = pol.ttl
        }

        if pol.Endpoint != "" {
            if endpoint, err = net.ResolveUDPAddr("udp4", pol.Endpoint); err != nil {
                return err
            }
        }

        switch pol.Egress {
        case "LOCAL"  : entry.Action.egress = e.local.netio
        case "TUNNEL" : entry.Action.egress = e.tunnel.netio
        default       : entry.Action.egress = nil
        }

        entry.Action.endpoint = endpoint

        tmpPolicy = append(tmpPolicy, entry)
    }

    e.policy = tmpPolicy
    return nil
}

func (e *Engine) Lookup(pkt *Packet) (PolicyAction, bool) {

    for _, entry := range e.policy {
        if (entry.Match.dstSubnet != nil) && (!entry.Match.dstSubnet.Contains(pkt.GetDestinationIPv4())) {
            continue
        }

        if (entry.Match.srcSubnet != nil) && (!entry.Match.srcSubnet.Contains(pkt.GetSourceIPv4())) {
            continue
        }

        return entry.Action, true
    }

    return PolicyAction{}, false
}


func (e *Engine) DumpPolicies() {
    var output string

    Print("Engine policies:")

    for _, pol := range e.policy {

        output = " "

        if pol.Match.srcSubnet != nil {
            output = output + pol.Match.srcSubnet.String()
        } else {
            output = output + "*"
        }

        output = output + " "

        if pol.Match.dstSubnet != nil {
            output = output + pol.Match.dstSubnet.String()
        } else {
            output = output + "*"
        }

        output = output + " ==> "

        if pol.Action.endpoint != nil {
            output = output + pol.Action.endpoint.String()
        } else {
            output = output + "*"
        }

       output = " " + output + strconv.Itoa(pol.TimeToLive)

       log.Println(output)
    }
}


func (e *Engine) PrintCounters() {

    Print("Engine counters:")

    for _, entry := range e.links {
        //log.Println("Link", entry.metadata.Name)
        log.Println("\tReceived:\t", entry.counters.Received)
        log.Println("\tSent:\t\t", entry.counters.Sent)
        log.Println("\tDropped:\t", entry.counters.Dropped)
        log.Println("\tUnsupported:\t", entry.counters.UnSupported)
        log.Println("\tError Receive:\t", entry.counters.ErrReceive)
        log.Println("\tError Send:\t", entry.counters.ErrSend)
    }
}
