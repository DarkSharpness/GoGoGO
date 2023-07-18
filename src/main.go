package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
)

// Whether to enable TLS hijack
const plays_genshin_impact = false

func main() {
	fmt.Println("Hello World!")
	Socks5_Link(":8080")
}

/* Process the connection. */
func Process(client net.Conn) {
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

	// Read first.
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

	// Check whether there is 0x00 method.
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
		if plays_genshin_impact {
			return TLS_Connection(client, atyp, addr)
		} else {
			return TCP_Connection(client, atyp, addr)
		}
	} else if cmd == 3 {
		return UDP_Connection(client, atyp, addr)
	} else {
		return errors.New("This should never happen...... God knows!")
	}
}

func Socks5_Link(addr string) {
	if plays_genshin_impact {
		cert, err := tls.LoadX509KeyPair("localhost.crt", "localhost.key")
		if err != nil {
			fmt.Println("Proxy: loadkey:", err)
			return
		}

		config := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		server, err := tls.Listen("tcp", addr, config)
		if err != nil {
			fmt.Println("Proxy: listen:", err)
			return
		}
		defer server.Close()

		for {
			client, err := server.Accept()
			if err != nil {
				fmt.Println("Proxy: accept:", err)
				return
			}
			go Process(client)
		}
	} else {
		server, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Println("Listen failed:", err)
			return
		}
		defer server.Close()

		for {
			client, err := server.Accept()
			if err != nil {
				fmt.Println("Accept failed:", err)
				continue
			}
			go Process(client)
		}
	}
}
