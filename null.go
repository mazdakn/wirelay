package main

type Null struct {
}

func (null *Null) Init() (error) {
    return nil
}

func (null *Null) Close() (error) {
    return nil
}

func (null *Null) Receive(pkt *Packet) (error) {
    return nil
}

func (null *Null) Send(pkt *Packet) (error) {
    pkt = nil
    return nil
}



