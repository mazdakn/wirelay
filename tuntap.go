package main

import (
    "water"
)

type TunTap struct {
    device *water.Interface
    Name string
}

func (iface *TunTap) Init() (error) {
    var err error

    config := water.Config{}
    config.Name = iface.Name

    config.DeviceType = water.TUN

    if iface.device, err = water.New(config); err != nil {
        return err
    }

    return nil
}

func (iface *TunTap) Close() (error) {
    iface.device.Close()
    return nil
}

func (iface *TunTap) Receive(pkt *Packet) (error) {
    var err error
    var n   int

    if n, err = iface.device.Read(pkt.Data); err != nil {
        return err
    }

    pkt.Size = uint16(n)

    return nil
}

func (iface *TunTap) Send(pkt *Packet) (error) {
    iface.device.Write(pkt.Data[:pkt.Size])
    return nil
}
