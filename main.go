package main

import (
	"log"
	"os"

	"github.com/felipetojal/broadcast-server/logger"
	"github.com/felipetojal/broadcast-server/server"
)

func main() {
	l := logger.NewLogger(os.Stdout)
	s := server.NewServer("127.0.0.1", "3000", l)

	if err := server.Start(s); err != nil {
		log.Fatal(err)
	}
}
