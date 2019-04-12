package main

type NetIO interface {
    Init() (error)
    Close() (error)
    Receive(*Packet) (error)
    Send(*Packet) (error)
}

const (
    NETIO_LOCAL    uint8 = 0
    NETIO_TUNNEL   uint8 = 1
    NETIO_DROP     uint8 = 2
    NETIO_MAX      uint8 = 3
)
