package main

import (
    "fmt"
    "os"
)

func main() {

    var conf Configuration
    if err := conf.Init(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    var engine Engine
    engine.conf = conf
    if err := engine.Init(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    engine.Start()
}
