package main

import (
	"fmt"
	"net"
)

func main() {
	// 1、与服务端建立连接
	conn, err := net.Dial("tcp", "127.0.0.1:1080")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}
	fmt.Println("Connection success")
	conn.Close()
}
