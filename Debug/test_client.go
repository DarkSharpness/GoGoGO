package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	// 1、与服务端建立连接
	conn, err := net.Dial("tcp", "127.0.0.1:1080")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}

	// 2、使用 conn 连接进行数据的发送和接收
	for {
		// 向客户端发送信息
		if !send_message(conn) {
			return
		}
		if !receive_message(conn) {
			return
		}
	}
}

/* Send one message. */ 
func send_message(conn net.Conn) bool {
	var s string
	fmt.Scanln(s)
	s = strings.TrimSpace(s)
	if strings.ToUpper(s) == "QUIT" {
		return false
	}
	_, err := conn.Write([]byte(s))
	if err != nil {
		fmt.Printf("send failed, err:%v\n", err)
		return false
	}
	return true
}

/* Receive one message. */
func receive_message(conn net.Conn) bool {
	var buf [1024]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		fmt.Printf("read failed:%v\n", err)
		return false
	}
	fmt.Printf("收到服务端回复:%v\n", string(buf[:n]))
	return true
}
