package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

// Main function for TCP
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
		var code = TCP_Error_Parse(err)
		client.Write([]byte{0x05, code, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return errors.New("dial failure: " + err.Error())
	}

	// IP string
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

	// Forward 2 connection
	fmt.Println("Connection Success!")
	go Forward_Client(client, target)
	go Forward_Target(client, target)
	return nil
}

// Parse a TCP Address
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

// Read from a TCP connection
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

func TCP_Error_Parse(err error) byte {
	str := err.Error()
	// error string shortcut
	if strings.Contains(str, "no route") {
		return 0x03
	} else if strings.Contains(str, "lookup") {
		return 0x04
	} else if strings.Contains(str, "network is unreachable") {
		return 0x03
	} else if strings.Contains(str, "name resolution") {
		return 0x04
	} else if strings.Contains(str, "refused") {
		return 0x05
	} else {
		// This shoud never happen
		fmt.Println("DEBUG || This should never happen......Fuck!")
		return 0x00
	}
}
