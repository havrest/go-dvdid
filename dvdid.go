package main

import (
    "log"
    "github.com/havrest/go-dvdid/dvdid"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatalf("Usage: %s <Volume path>", os.Args[0])
    }

    log.Printf("Analysing volume: %s", os.Args[1])

    discId, err := dvdid.ComputeDVDId(os.Args[1])

    if err != nil {
        log.Fatalf("Err: %s", err.Error())
    } else {
        log.Printf("DiscId mymovies.dk style: %X-%X", discId[0:4], discId[4:])
        log.Printf("DiscId pydvdid style: %x", discId)
    }
}
