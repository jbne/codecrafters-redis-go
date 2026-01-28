package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	address := "127.0.0.1"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	fmt.Printf("Start listening on %s\n", endpoint)
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		fmt.Printf("Failed to bind to %s: %s", endpoint, err)
		os.Exit(1)
	}

	fmt.Println("Waiting for client connection...")
	conn, err := listener.Accept()
	listener.Close()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	fmt.Println("Client connected!")
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Read err: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n", buf)
		conn.Write([]byte("+PONG\r\n"))
		if err != nil {
			fmt.Printf("Write err: %v\n", err)
			os.Exit(1)
		}
	}
}
