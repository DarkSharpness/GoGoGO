package main

import (
	"errors"
	"fmt"
	"io"
	"net"
)

func main() {
	server, err := net.Listen("tcp", ":1080")
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
	defer client.Close()

	// remoteAddr := client.RemoteAddr().String()
	// fmt.Printf("Connection from %s\n", remoteAddr)

	// client.Write([]byte("Hello world!"))
	err := Socks5_Auth(client)
	if err != nil {
		fmt.Println("Authorization failed: %v", err)
		return
	}

	target, err := Socks5_Connect(client)
	if err != nil {
		fmt.Println("Connect failed: %v", err)
		return
	}

	Socks5_Forward(client, target)
}

func Socks5_Auth(client net.Conn) error {
	var buf [512]byte

	// Read in first 2 byte of header information.
	n, err := client.Read(buf[:2])
	if n != 2 {
		return errors.New("Header Error " + err.Error())
	}

	// Check the version (Socks 5 only)
	ver, Nmethod := int(buf[0]), int(buf[1])
	if ver != 0x05 {
		return errors.New("Invalid Version!")
	}

	// Read in the method data.
	n, err = client.Read(buf[:Nmethod])
	if Nmethod != n {
		return errors.New("Method length mismatch!")
	}

	// We choose 0x00 method
	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("Write error" + err.Error())
	}

	return nil
}

func Socks5_Connect(client net.Conn) (net.Conn, error) {
	var buf [512]byte

	// Read in first 2 byte of header information.
	n, err := client.Read(buf[:4])
	if n != 4 {
		return nil, errors.New("Header Error " + err.Error())
	}

	ver, cmd, rsv, atyp := int(buf[0]), int(buf[1]), int(buf[2]), int(buf[3])
	if ver != 0x05 || cmd != 1 || rsv != 0x00 {
		return nil, errors.New("Invalid ver/cmd/rsv!")
	}

	switch atyp {
	case 0x01:
		break

	case 0x03:
		break

	case 0x04:
		break

	default:
		return nil, errors.New("Invalid aytp!")
	}
}

func Socks5_Forward(client, target net.Conn) {
	go forward(client, target)
	go forward(target, client)
}

func forward(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
