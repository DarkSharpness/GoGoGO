package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

func main() {
	fmt.Println("Hello World!")
	server, err := net.Listen("tcp", ":8080")
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
	err := Socks5_Auth(client)
	if err != nil {
		fmt.Printf("Authorization failed: %v\n", err)
		return
	}

	err = Socks5_Connect(client)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		client.Close()
		return
	}

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

func Socks5_Connect(client net.Conn) error {
	var buf [260]byte

	// Read in first 4 byte of header information.
	_, err := io.ReadFull(client, buf[:4])
	if err != nil {
		return errors.New("Header Error " + err.Error())
	}

	// Check version,cmd and reservation
	ver, cmd, rsv, atyp := int(buf[0]), int(buf[1]), int(buf[2]), int(buf[3])

	if ver != 0x05 || (cmd != 1 && cmd != 3) || rsv != 0x00 {
		client.Write([]byte{0x05, 0x07, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return errors.New("invalid ver/cmd/rsv!")
	}

	// Get the full string and the buffer first
	user_addr, err := TCP_Address_Read(client, atyp)
	if err != nil {
		if err.Error() == "invalid address!" {
			client.Write([]byte{0x05, 0x08, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
		return err
	}
	addr := TCP_Address_Parse(user_addr, atyp)

	if cmd == 1 {
		return TCP_Connection(client, atyp, addr)
	} else if cmd == 3 {
		return UDP_Connection(client, atyp, addr)
	} else {
		return nil
	}
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
