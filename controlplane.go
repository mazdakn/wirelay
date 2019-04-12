package main

type ControlPlane struct {
    conf *Configuration
    port NetworkPort
}

func (c *ControlPlane) Init() (error) {
    var err error
    c.port.netio = &UDPSocket{LocalSocket: c.conf.content.Control}
    err = c.port.netio.Init()
    Fatal(err)

    return nil
}


