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

type Counters struct {
    Received    uint32
    Sent        uint32
    Dropped     uint32
    ErrReceive  uint32
    ErrSend     uint32
    UnSupported uint32
}

type NetworkPort struct {
    netio       NetIO
    counters    Counters
}
