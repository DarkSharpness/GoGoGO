package main

import (
	"fmt"
	"io"
	"net"
)

func main() {
	fmt.Println("Hello World!")
	Socks5_Link(":7070", ":8080")
}

func Socks5_Link(client_addr, proxy_addr string) {
	server, err := net.Listen("tcp", client_addr)
	if err != nil {
		fmt.Println("Listen failed:", err)
		return
	}
	defer server.Close()
	for {
		client, err := server.Accept()
		if err != nil {
			fmt.Println("Accept failed:", err)
			continue
		}
		go Process(client, proxy_addr)
	}
}

func Process(client net.Conn, addr string) {
	proxy, err := net.Dial("tcp", addr)
	if err != nil {
		client.Close()
		fmt.Println("Fail to connect to proxy server!")
		return
	}
	go Proxy_Forward(client, proxy)
	go Proxy_Forward(proxy, client)
}

// The most basic forward function.
func Proxy_Forward(client, target net.Conn) {
	defer client.Close()
	defer target.Close()
	io.Copy(client, target)
}
