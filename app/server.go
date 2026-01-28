package main

import (
	"fmt"
	"net"
	"os"
)

func worker(conn net.Conn) {
	fmt.Printf("Client connected! conn: %s\n", conn.RemoteAddr())
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Read err: %v\n", err)
			return
		}

		fmt.Printf("%s: %s\n", conn.RemoteAddr(), buf)
		_, err = conn.Write([]byte("+PONG\r\n"))
		if err != nil {
			fmt.Printf("Write err: %v\n", err)
			return
		}
	}
}

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

	for {
		fmt.Println("Waiting for client connection...")
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go worker(conn)
	}
}
