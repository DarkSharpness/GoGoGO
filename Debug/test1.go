// Server
package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	listen, err := net.Listen("tcp", "0.0.0.0:1919")
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}
	defer listen.Close()

	for {
		conn, _ := listen.Accept()
		go process(conn)
	}
}

func process(conn net.Conn) {
	defer conn.Close()

	fmt.Print(conn, " is successfully set up!\n")

	reader := bufio.NewReader(conn)
	var buf [512]byte


	for {
		reader := bufio.NewReader(conn)

		var buf [512]byte
		n, err := reader.Read(buf[:])
		recv := string(buf[:n])

	}
}
