package main

type Drop struct {
}

func (d *Drop) Init() (error) {
    return nil
}

func (d *Drop) Close() (error) {
    return nil
}

func (d *Drop) Receive(pkt *Packet) (error) {
    return nil
}

func (d *Drop) Send(pkt *Packet) (error) {
    return nil
}
