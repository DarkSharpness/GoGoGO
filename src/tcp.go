package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func TCP_Connection(client net.Conn, atyp int, addr string) error {
	// Custom debug message.
	if atyp == 0x03 {
		fmt.Printf("Website domain: %v\n", addr)
	} else if atyp == 0x01 {
		fmt.Printf("Website ipv4: %v\n", addr)
	} else {
		fmt.Printf("Website ipv6: %v\n", addr)
	}

	// Tries to dial the address
	target, err := net.Dial("tcp", addr)
	if err != nil {
		var code byte
		// error string shortcut
		str := err.Error()
		if strings.Contains(str, "no route") {
			code = 0x03
		} else if strings.Contains(str, "lookup") {
			code = 0x04
		} else if strings.Contains(str, "network is unreachable") {
			code = 0x03
		} else if strings.Contains(str, "name resolution") {
			code = 0x04
		} else if strings.Contains(str, "refused") {
			code = 0x05
		}
		client.Write([]byte{0x05, code, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return errors.New("dial failure: " + str)
	}

	IP := Parse_IP_Port(target.LocalAddr().String())
	if len(IP) == 6 { // ipv4 = 4 + 2
		atyp = 0x01
	} else { // ipv6 = 16 + 2
		atyp = 0x04
	}

	_, err = client.Write(append([]byte{0x05, 0x00, 0x00, byte(atyp)}, IP...))
	if err != nil {
		target.Close()
		return err
	}

	TCP_forward(client, target)
	return nil
}

func Parse_IP_Port(str string) []byte {
	ip_str, port_str, _ := net.SplitHostPort(str)
	ip := net.ParseIP(ip_str)
	port, _ := strconv.Atoi(port_str)
	return binary.BigEndian.AppendUint16(ip, uint16(port))
}

func TCP_forward(client, target net.Conn) {
	fmt.Println("Connection Success!")
	// Forward 2 connection
	go Forward(client, target)
	go Forward(target, client)
}

func Forward(writer, reader net.Conn) {
	defer writer.Close()
	defer reader.Close()
	buf := make([]byte, 32*1024)
	for {
		nr, err := reader.Read(buf[:32*1024])
		if nr > 0 {
			nw, er := writer.Write(buf[0:nr])
			if nr < nw || er != nil {
				return
			}
		}
		fmt.Println("DEBUG ||", string(buf[:64]))
		if err != nil {
			return
		}
	}
}
