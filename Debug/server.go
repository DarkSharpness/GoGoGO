package main

import (
	"bufio"
	"fmt"
	"net"
)

func process(conn net.Conn) {
	// 处理完关闭连接
	defer conn.Close()

	reader := bufio.NewReader(conn)
	var buf [512]byte
	n, err := reader.Read(buf[:])

	if err != nil {
		fmt.Printf("read from conn failed, err:%v\n", err)
		return
	}

	recv := string(buf[:n])
	if recv[0] != 0x05 {
		fmt.Printf("Version mismatch! Try socks5 instead!\n")
		return
	} else if recv[1] == 0x00 {
		fmt.Printf("No available methods!\n")
		return
	} else if int(recv[1]) != n {
		fmt.Printf("Method length mismatch!")
	}

	// 将接受到的数据返回给客户端
	_, err = conn.Write([]byte("ok"))
	if err != nil {
		fmt.Printf("write from conn failed, err:%v\n", err)
		return
	}
}

func main() {
	// 建立 tcp 服务
	listen, err := net.Listen("tcp", "0.0.0.0:1919")
	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}
	defer listen.Close()
	for {
		// 等待客户端建立连接
		conn, err := listen.Accept()
		if err != nil {
			fmt.Printf("accept failed, err:%v\n", err)
			continue
		}
		// 启动一个单独的 goroutine 去处理连接
		go process(conn)
	}
}
