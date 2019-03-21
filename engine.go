package main

import (
    "log"
    "sync"
    "io/ioutil"
    "encoding/json"
    "errors"
)

const (
    NetIO_ANY    = 0
    NetIO_NULL   = 1
    NetIO_LOCAL  = 2
    //NetIO_TUNNEL = 3
    NetIO_MAX    = 10
)

type NetIO interface {
    Init() (error)
    Close() (error)
    Receive(*Packet) (error)
    Send(*Packet) (error)
}

var (
    ErrEngineConfigFile = errors.New("Error in config file")
)

type EngineConfiguration struct {
    Name     string `json:"name"`
    ID       uint8  `json:"id"`
    Type     string `json:"type"`
    Address  string `json:"address"`
    Key      string `json:"key"`
    Pubkey   string `json:"pubkey"`
    Policies []PolicyEntryFile `json:"policy"`
    //Policy  string  `json:"policy"`
    //TunTap  string `json:"tuntap"`
    //Tunnel  string `json:"tunnel"`
}

type EngineEntry struct {
    policy Policy
    netio  NetIO
}

type Engine struct {
    Configfile string
    config     []EngineConfiguration
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

        pkt.Endpoint = action.endpoint
        if err := e.interfaces[action.egress].netio.Send(&pkt); err != nil {
            log.Println (err)
        }
        //if err := action.Perform(&pkt); err != nil {
        //    //TODO: should change to counters
        //    continue
        //}
    }
}

//nc (e *Engine) ThreadRx()

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
        /*log.Println("Startin tunnel")
        go e.Forward(e.interfaces[NetIO_TUNNEL], &waitGroup)
        waitGroup.Add(1)

        log.Println("Starting tuntap")
        go e.Forward(e.interfaces[NetIO_LOCAL], &waitGroup)
        waitGroup.Add(1)*/

	waitGroup.Wait()
	log.Println ("Shuting down")
}

func (e *Engine) Init () (error) {
    var entry EngineEntry

    if e.Configfile == "" {
        return ErrEngineConfigFile
    }

    if err := e.ReadJSON(); err != nil {
        return ErrEngineConfigFile
    }

    e.interfaces = make([]EngineEntry, NetIO_MAX)

    for _, netio := range e.config {

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
        //e.Interfaces = append(e.interfaces, entry)
    }

    /*
    //  is it necessary?
    interfaces[NetIO_ANY] = nil

    //interfaces[NetIO_NULL] = &Null{}
    err := e.interfaces[NetIO_NULL].Init(); err != nil {
        return err
    }

    e.interfaces[NetIO_TUNNEL] = &Tunnel{LocalSocket: e.config.Tunnel}
    if err := e.interfaces[NetIO_TUNNEL].Init(); err != nil {
        return err
    }

    e.interfaces[NetIO_LOCAL] = &TunTap{Name: e.config.TunTap}
    if err := e.interfaces[NetIO_LOCAL].Init(); err != nil {
        return err
    }

    e.policy.configFile = e.config.Policy
    e.policy.Init()
    if err := e.policy.CompilePoliciesJSON(); err != nil {
        return err
    }*/

    //t := netio.UDPTunnel{LocalSocket: e.config.Tunnel, Endpoint: "192.168.1.10:9000"}
    //t1 := netio.UDPTunnel{LocalSocket: e.config.Tunnel, Endpoint: "192.168.1.20:9000"}
    //log.Println (t, t1)

    return nil
}

func (e *Engine) ReadJSON() (error) {
    var bytes []byte
    var err error

    if bytes, err = ioutil.ReadFile(e.Configfile); err != nil {
        return err
    }

    if err := json.Unmarshal(bytes, &e.config); err != nil {
        return err
    }

    return nil
}

func (e *Engine) SaveJSON() (error) {
    var bytes []byte
    var err error

    if bytes, err = json.Marshal(&e.config); err != nil {
        return err
    }

    if err = ioutil.WriteFile(e.Configfile, bytes, 0644); err != nil {
        return err
    }

    return nil
}
