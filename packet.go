package main

import (
    "net"
)

type Packet struct {
    Data        []byte
    Endpoint    *net.UDPAddr
}

// TODO: Add IPv6 support

func (pkt *Packet) IsIPv4() bool {
    return (pkt.Data[0] >> 4) == 4
}

func (pkt *Packet) GetSourceIPv4() net.IP {
    return net.IPv4(pkt.Data[12], pkt.Data[13], pkt.Data[14], pkt.Data[15])
}

func (pkt *Packet) GetDestinationIPv4() net.IP {
    return net.IPv4(pkt.Data[16], pkt.Data[17], pkt.Data[18], pkt.Data[19])
}
