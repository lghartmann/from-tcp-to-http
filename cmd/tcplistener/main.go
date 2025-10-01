package main

import (
	"fmt"
	"log"
	"net"

	"github.com/lghartmann/from-tcp-to-http/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatalf("unable to open file: %s", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("error listening", "error=", err)
		}

		r, err := request.RequestFromReader(conn)
		if err != nil {
			log.Fatal("error", "error", err)
		}

		fmt.Printf("Request line:\n")
		fmt.Printf("- Method: %s\n", r.RequestLine.Method)
		fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", r.RequestLine.HttpVersion)
	}

}
