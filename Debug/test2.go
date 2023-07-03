// Client
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// 1、与服务端建立连接
	conn, err := net.Dial("tcp", "127.0.0.1:1919")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}
	fmt.Println("Connetction build!")

	input := bufio.NewReader(os.Stdin)

	for {
		s, _ := input.ReadString('\n')
		s = strings.TrimSpace(s)
		if strings.ToUpper(s) == "QUIT" {
			return
		}

		conn.Write([]byte(s))

		// 从服务端接收回复消息
		var buf [512]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Printf("read failed:%v\n", err)
			return
		}

		fmt.Printf("收到服务端回复:%v\n", string(buf[:n]))
	}
}
