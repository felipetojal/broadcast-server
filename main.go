package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/felipetojal/broadcast-server/client"
	"github.com/felipetojal/broadcast-server/logger"
	"github.com/felipetojal/broadcast-server/server"
)

var (
	addr *string
	srv  *bool
	clt  *bool
)

func init() {
	addr = flag.String("addr", "", "insert the ip address (include port)")
	srv = flag.Bool("server", false, "set true to run node as server")
	clt = flag.Bool("client", false, "set true to run node as client")
	flag.Parse()
}

func main() {
	bothTrue := (*clt) && (*srv)
	bothFalse := !(*clt) && !(*srv)
	// Checking flag inconsistencies.
	if bothFalse || bothTrue {
		log.Fatal("must set only one option true")
	}

	switch {
	case (*srv):
		l := logger.NewLogger(os.Stdout)
		split := strings.Split((*addr), ":")
		if len(split) > 2 {
			log.Fatal("error spliting address")
		}
		host := split[0]
		port := split[1]
		s := server.NewServer(host, port, l)

		if err := server.Start(s); err != nil {
			log.Printf("addr: %s", (*addr))
			log.Printf("server: %v", *srv)
			log.Printf("server: %v", *clt)
			log.Fatal("error starting server")
		}
	case (*clt):
		client.EstablishConn(*addr)
	}
}
