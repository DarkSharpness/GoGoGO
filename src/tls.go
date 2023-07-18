package main

import (
	"errors"
	"net"
)

func TLS_Connection(client net.Conn, atyp int, addr string) error {
	defer client.Close()
	server, err := net.Dial("tcp", addr)
	if err != nil {
		var code = TCP_Error_Parse(err)
		client.Write([]byte{0x05, code, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return errors.New("dial failure: " + err.Error())
	}

	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go Forward_Client(client, server)
	go Forward_Target(client, server)
	return nil
}
