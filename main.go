package main

import (
    "fmt"
)

func main() {
    var engine Engine

    if err := engine.Init(); err != nil {
        fmt.Println(err)
        return
    }

    engine.Start()
}
