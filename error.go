package main

import (
    "log"
    "os"
)

func Print(message string) {
        log.Println(message)
}

func Exit(message string) {
    log.Println(message)
    os.Exit(1)
}

func Log(err error) {
    if (err != nil) {
        log.Println(err)
    }
}

func Fatal(err error) {
    if (err != nil) {
        log.Fatal(err)
    }
}
