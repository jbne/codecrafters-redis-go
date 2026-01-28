package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	address := "127.0.0.1"
	port := "6379"
	endpoint := fmt.Sprintf("%s:%s", address, port)

	fmt.Printf("Dialing %s\n", endpoint)
	conn, err := net.Dial("tcp", "127.0.0.1:6379")

	if err != nil {
		fmt.Printf("Failed to dial %s\n", endpoint)
		os.Exit(1)
	}

	fmt.Printf("Successfully dialed %s!\n", endpoint)
	defer conn.Close()

	scanner := bufio.NewScanner(os.Stdin)
	promptString := "$ "
	buf := make([]byte, 1024)

	for {
		os.Stdout.WriteString(promptString)

		scanner.Scan()
		input := scanner.Text()

		_, err := conn.Write([]byte(input))

		if err != nil {
			fmt.Printf("Write err: %v\n", err)
			os.Exit(1)
		}

		_, err = conn.Read(buf)
		if err != nil {
			fmt.Printf("Read err: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.WriteString(string(buf))
	}
}
