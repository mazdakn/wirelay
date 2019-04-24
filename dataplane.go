package main

import (
    "log"
    "sync"
)

type DataPlane struct {
    conf    *Configuration
    ports   [NETIO_MAX]NetworkPort
    rules   Policy
}

/* Initilizing the Wirelay Engine
   1 - Create and initialize the slice of interfaces from the Configuration
   2 - Create Policy object for each interface
   3 - Setup the signal handler
*/
func (d *DataPlane) Init() (error){
    var err   error

    // Create local TUN interface
    d.ports[NETIO_LOCAL].netio = &TunTap{Name: d.conf.content.Name}
    if err = d.ports[NETIO_LOCAL].netio.Init(); err != nil {
        return err
    }

    d.ports[NETIO_TUNNEL].netio = &UDPSocket{LocalSocket: d.conf.content.Data}
    if err = d.ports[NETIO_TUNNEL].netio.Init(); err != nil {
        return err
    }

    d.ports[NETIO_DROP].netio = &Drop{}
    if err = d.ports[NETIO_DROP].netio.Init(); err != nil {
        return err
    }

    for _, pol := range d.conf.content.Policies {
        if err = d.rules.CompilePolicy(pol); err != nil {
            log.Println(err)
        }
    }

    return nil
}

func (d *DataPlane) Start(waitGroup *sync.WaitGroup) (int) {

    go d.Forward(&d.ports[NETIO_LOCAL], waitGroup)
    go d.Forward(&d.ports[NETIO_TUNNEL], waitGroup)
    waitGroup.Add(2)

    return 2
}

func (d *DataPlane) Forward(dev *NetworkPort, waitGroup *sync.WaitGroup) {
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

        if action, found = d.rules.Lookup(&pkt); !found {
            dev.counters.Dropped++
            continue
        }

        pkt.Endpoint = action.endpoint
        if err := d.ports[action.egress].netio.Send(&pkt); err != nil {
            dev.counters.ErrSend++
            continue
        }

        dev.counters.Sent++
    }
}

func (d *DataPlane) PrintCounters() {
    names := []string{"Local", "Tunnel", "Drop"}
    Print("Engine counters:")

    for index, entry := range d.ports {
        log.Println(names[index] + ":")
        log.Println("\tReceived:\t", entry.counters.Received)
        log.Println("\tSent:\t\t", entry.counters.Sent)
        log.Println("\tDropped:\t", entry.counters.Dropped)
        log.Println("\tUnsupported:\t", entry.counters.UnSupported)
        log.Println("\tError Receive:\t", entry.counters.ErrReceive)
        log.Println("\tError Send:\t", entry.counters.ErrSend)
    }
}
