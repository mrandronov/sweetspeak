package main

import (
        log "sweetspeak/logging"
        "sweetspeak/server"
)


func main() {
        log.SetGlobalFile("sweetspeak-server.log")

        ss := server.New()
        ss.Start()
}
