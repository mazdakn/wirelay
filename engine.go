package main

import (
    "log"
    "sync"
    "io/ioutil"
    "encoding/json"
    "errors"
)

const (
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
    }

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
