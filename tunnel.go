package main

import (
    "net"
    "errors"
)

var (
    ErrTunnelSocketRX = errors.New("Error in reading from udp socket")
    ErrTunnelSocketClosed = errors.New("udp socket closed")
    ErrTunnelSocketNotReady = errors.New ("Socket is not ready yet")
    ErrTunnelSocketListener = errors.New("Invalid local address")
)

type Tunnel struct {
    listener    *net.UDPConn
    local       *net.UDPAddr
    LocalSocket string
}

// initialize udp tunnel
func (t *Tunnel) Init() (error) {
    var err error

    t.listener = nil
    t.local = nil

    // resolve local address -> ip:port    
    if t.local, err = net.ResolveUDPAddr ("udp4", t.LocalSocket); err != nil {
        return err
    }

    // listen on local ip:port
    if t.listener, err = net.ListenUDP ("udp4", t.local); err != nil {
        return err
    }

    return nil
}

// close udp tunnel
func (t *Tunnel) Close() (error) {
    t.listener.Close()
    return nil
}

// receive packet from remote peer
func (t *Tunnel) Receive(pkt *Packet) (error) {
    var err error
    var n int

    if t.listener == nil {
        return ErrTunnelSocketNotReady
    }

    // read packet from udp tunnel
    if n, _, err = t.listener.ReadFromUDP (pkt.Data); err != nil {
        return ErrTunnelSocketRX
    }

    if n == 0 {
        return ErrTunnelSocketClosed
    }

    pkt.Size = uint16(n)

    return nil
}

// send packet to remote peer
func (t *Tunnel) Send(pkt *Packet) (error) {
    var err error

    if t.listener == nil {
       return ErrTunnelSocketNotReady
    }

    // send packet to udp tunnel
    if _, err = t.listener.WriteToUDP (pkt.Data[:pkt.Size], pkt.Endpoint); err != nil {
        return err
    }

    return nil
}
