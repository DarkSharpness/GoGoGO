package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

func main() {
	server, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Listen failed: %v\n", err)
		return
	}
	defer server.Close()

	for {
		client, err := server.Accept()
		if err != nil {
			fmt.Println("Accept failed: %v", err)
			continue
		}
		go process(client)
	}
}

/* Process the connection. */
func process(client net.Conn) {
	err := Socks5_Auth(client)
	if err != nil {
		fmt.Printf("Authorization failed: %v\n", err)
		return
	}

	target, err := Socks5_Connect(client)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		client.Close()
		return
	}

	Socks5_Forward(client, target)
}

func Socks5_Auth(client net.Conn) error {
	var buf [260]byte

	_, err := io.ReadFull(client, buf[:2])
	if err != nil {
		return errors.New("read Error: " + err.Error())
	}

	// Check the version (Socks 5 only)
	ver, Nmethod := int(buf[0]), int(buf[1])
	if ver != 0x05 {
		return errors.New("invalid Version!")
	}

	// Read in the method data.
	_, err = io.ReadFull(client, buf[:Nmethod])
	if err != nil {
		return errors.New("read Error: " + err.Error())
	}

	flag := true
	for i := 0; i != Nmethod; i++ {
		if buf[i] == 0x00 {
			flag = false
			break
		}
	}
	if flag == true {
		client.Write([]byte{0x05, 0xff})
		return errors.New("no acceptable methods.")
	}

	// We choose 0x00 method
	_, err = client.Write([]byte{0x05, 0x00})
	if err != nil {
		return errors.New("write error: " + err.Error())
	}

	return nil
}

func Socks5_Connect(client net.Conn) (net.Conn, error) {
	var buf [260]byte

	// Read in first 4 byte of header information.
	_, err := io.ReadFull(client, buf[:4])
	if err != nil {
		return nil, errors.New("Header Error " + err.Error())
	}

	// Check version,cmd and reservation
	ver, cmd, rsv, atyp := int(buf[0]), int(buf[1]), int(buf[2]), int(buf[3])

	if ver != 0x05 || cmd != 1 || rsv != 0x00 {
		client.Write([]byte{0x05, 0x07, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return nil, errors.New("invalid ver/cmd/rsv!")
	}

	// Get the full string and the buffer first
	addr_arr, err := TCP_Address_Read(client, atyp)
	if err != nil {
		if err.Error() == "invalid address!" {
			client.Write([]byte{0x05, 0x08, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
		return nil, err
	}
	addr := TCP_Address_Parse(addr_arr, atyp)

	// Custom debug message.
	if atyp == 0x03 {
		fmt.Printf("Website domain: %v\n", addr)
	} else if atyp == 0x01 {
		fmt.Printf("Website ipv4: %v\n", addr)
	} else {
		fmt.Printf("Website ipv6: %v\n", addr)
	}

	// Tries to dial the address
	dest, err := net.Dial("tcp", addr)
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
		return nil, errors.New("dial failure: " + str)
	}

	IP := Parse_IP_Port(dest.LocalAddr().String())
	if len(IP) == 6 { // ipv4 = 4 + 2
		atyp = 0x01
	} else { // ipv6 = 16 + 2
		atyp = 0x04
	}

	_, err = client.Write(append([]byte{0x05, 0x00, 0x00, byte(atyp)}, IP...))
	if err != nil {
		dest.Close()
		return nil, err
	}
	return dest, nil
}

func TCP_Address_Parse(buf []byte, atyp int) string {
	addr := ""
	switch atyp {
	case 0x01: // ipv4 case
		addr = fmt.Sprintf("%d.%d.%d.%d:%d", buf[0], buf[1], buf[2], buf[3],
			binary.BigEndian.Uint16(buf[4:6]))

	case 0x03: // domain case
		len := int(buf[0])
		addr = string(buf[1:1+len]) + fmt.Sprintf(":%v",
			binary.BigEndian.Uint16(buf[1+len:1+len+2]))

	case 0x04: // ipv6 case
		addr = "["
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[0:2]))
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[2:4]))
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[4:6]))
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[6:8]))
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[8:10]))
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[10:12]))
		addr += fmt.Sprintf("%x:", binary.BigEndian.Uint16(buf[12:14]))
		addr += fmt.Sprintf("%x]", binary.BigEndian.Uint16(buf[14:16]))
		addr += fmt.Sprintf(":%d", binary.BigEndian.Uint16(buf[16:18]))
	}
	return addr
}

func TCP_Address_Read(client net.Conn, atyp int) ([]byte, error) {
	var buf [260]byte
	var err error

	// Check the atyp first
	switch atyp {
	case 0x01: // ipv4 case
		_, err = io.ReadFull(client, buf[:6])
		if err != nil {
			return nil, errors.New("read error!" + err.Error())
		}

		return buf[:6], nil

	case 0x03: // domain case
		_, err = io.ReadFull(client, buf[:1])
		if err != nil {
			return nil, errors.New("read error!" + err.Error())
		}
		len := int(buf[0])
		_, err = io.ReadFull(client, buf[1:1+len+2])
		if err != nil {
			return nil, errors.New("read error!" + err.Error())
		}

		return buf[:1+len+2], nil

	case 0x04: // ipv6 case
		_, err = io.ReadFull(client, buf[:18])
		if err != nil {
			return nil, errors.New("read error!" + err.Error())
		}

		return buf[:18], nil

	default:
		return nil, errors.New("invalid address!")
	}
}

func Socks5_Forward(client, target net.Conn) {
	fmt.Println("Connection Success!")
	// Forward 2 connection
	go Forward(client, target)
	go Forward(target, client)
}

func Forward(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}

func Parse_IP_Port(str string) []byte {
	IP_str, port_str, _ := net.SplitHostPort(str)
	IP := net.ParseIP(IP_str)
	port, _ := strconv.Atoi(port_str)
	return binary.BigEndian.AppendUint16(IP, uint16(port))
}
