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
    egress      *NetIO
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
    policy      []PolicyEntry
    counters    Counters
    name        string
}

type Engine struct {
    conf Configuration
    links []EngineEntry
}


/* Initilizing the Wirelay Engine
   1 - Create and initialize the slice of interfaces from the Configuration
   2 - Create Policy object for each interface
   3 - Setup the signal handler
*/
func (e *Engine) Init() {
    var entry EngineEntry
    var err   error

    err = e.conf.Init()
    Fatal(err)

    e.links = make([]EngineEntry, 0)

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

        entry.name = netio.Name

        e.links = append(e.links, entry)
    }

    for index, netio := range e.conf.content {

        err = e.CompilePolicies(index, netio.Policies)
        //g.Println(e.links[index].policy)
        Fatal(err)
    }

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

    for index, entry := range e.links {
        if entry.netio != nil{
            go e.Forward(index, entry, &waitGroup)
            waitGroup.Add(1)
        }
    }

	waitGroup.Wait()
	Print("Shuting down")
}


func (e *Engine) Forward(index int, entry EngineEntry, waitGroup *sync.WaitGroup) {
    var pkt Packet
    var action PolicyAction
    var found bool

   // TODO: set MTU and read correct number of bytes
    pkt.Data = make([]byte, 2000)

    defer waitGroup.Done()
    for {
        if err := entry.netio.Receive(&pkt); err != nil {
            entry.counters.ErrReceive++
            continue
        }

        e.links[index].counters.Received++

        if !pkt.IsIPv4() {
            entry.counters.UnSupported++
            continue
        }

        if action, found = e.Lookup(index, &pkt); !found {
            entry.counters.Dropped++
            continue
        }

        if action.egress == nil {
            entry.counters.Dropped++
            continue
        }

        pkt.Endpoint = action.endpoint
        if err := (*action.egress).Send(&pkt); err != nil {
            entry.counters.ErrSend++
            continue
        }

        e.links[index].counters.Sent++
    }
}


func (e *Engine) CompilePolicies(index int, policies []PolicyEntryFile) (error) {

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

        if index, found := e.FindNetIOByName(pol.Egress); found {
            entry.Action.egress = &(e.links[index].netio)
        } else {
            entry.Action.egress = nil
        }

        entry.Action.endpoint = endpoint

        tmpPolicy = append(tmpPolicy, entry)
    }

    e.links[index].policy = tmpPolicy
    return nil
}


func (e *Engine) FindNetIOByName(name string) (int, bool) {

    for index, item := range e.links {
        if item.name == name {
            return index, true
        }
    }

    return 0, false
}


func (e *Engine) Lookup(index int, pkt *Packet) (PolicyAction, bool) {

    for _, entry := range e.links[index].policy {
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

    for _, link := range e.links {

        Print("Link " +  link.name)

        for _, pol := range link.policy {

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

            output = output + " " //+ item.Action.egress + " "

            if pol.Action.endpoint != nil {
                output = output + pol.Action.endpoint.String()
            } else {
                output = output + "*"
            }

            output = output + " "
            output = output + strconv.Itoa(pol.TimeToLive)

            log.Println(output)
        }
    }
}


func (e *Engine) PrintCounters() {

    Print("Enging counters:")

    for _, entry := range e.links {
        log.Println("Link", entry.name)
        log.Println("\tReceived:\t", entry.counters.Received)
        log.Println("\tSent:\t\t", entry.counters.Sent)
        log.Println("\tDropped:\t", entry.counters.Dropped)
        log.Println("\tUnsupported:\t", entry.counters.UnSupported)
        log.Println("\tError Receive:\t", entry.counters.ErrReceive)
        log.Println("\tError Send:\t", entry.counters.ErrSend)
    }
}
