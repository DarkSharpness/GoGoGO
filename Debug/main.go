package main

import (
	"fmt"
	"net"
)

func main() {
	server, err := net.Listen("tcp", ":1080")
	if err != nil {
		fmt.Printf("Listen failed: %v\n", err)
		return
	}
	defer server.Close()

	for {
		client, err := server.Accept()
		if err != nil {
			fmt.Printf("Accept failed: %v", err)
			continue
		}
		go process(client)
	}
}

/* Process the connection. */
func process(client net.Conn) {
	defer client.Close()

	remoteAddr := client.RemoteAddr().String()
	fmt.Printf("Connection from %s\n", remoteAddr)

	client.Write([]byte("Hello world!"))
}

