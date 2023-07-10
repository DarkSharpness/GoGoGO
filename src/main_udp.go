package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
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

func Socks5_Connect(client_tcp net.Conn) error {
	var buf [260]byte

	// Read in first 4 byte of header information.
	_, err := io.ReadFull(client_tcp, buf[:4])
	if err != nil {
		return errors.New("Header Error " + err.Error())
	}

	// Check version,cmd and reservation
	ver, cmd, rsv, atyp := int(buf[0]), int(buf[1]), int(buf[2]), int(buf[3])
	if ver != 0x05 || cmd != 3 || rsv != 0x00 {
		client_tcp.Write([]byte{0x05, 0x07, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return errors.New("invalid ver/cmd/rsv!")
	}

	// Get the full string and the buffer first
	user_addr, err := TCP_Address_Read(client_tcp, atyp)
	if err != nil {
		if err.Error() == "invalid address!" {
			client_tcp.Write([]byte{0x05, 0x08, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
		return err
	}

	// Get client address information.
	client_addr, err := net.ResolveUDPAddr("udp",
		TCP_Address_Parse(user_addr[:], atyp))
	if err != nil {
		if err.Error() == "invalid address!" {
			client_tcp.Write([]byte{0x05, 0x08, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
		return err
	}
	fmt.Println("DEBUG || Client udp ip :", user_addr)
	fmt.Println("DEBUG || Client udp ip :", client_addr)

	// The server for client listening.
	client_udp, err := net.ListenUDP("udp", nil)
	if err != nil {
		client_tcp.Write([]byte{0x05, 0x01, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return err
	}

	// The server for remote listening.
	remote_udp, err := net.ListenUDP("udp", nil)
	if err != nil {
		client_udp.Close()
		client_tcp.Write([]byte{0x05, 0x01, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return err
	}

	// Local address information.
	fmt.Println("DEBUG || Local udp ip:", client_udp.LocalAddr())
	local_addr := Parse_IP_Port(client_udp.LocalAddr().String())
	// local_addr = []byte{0x7f, 0x00, 0x00, 0x01, 0x23, 0x28}
	fmt.Println("DEBUG || Local udp ip:", local_addr)

	// Atyo type
	var atyp_new byte
	if len(local_addr) == 6 { // ipv4 = 4 + 2
		atyp_new = 0x01
	} else { // ipv6 = 16 + 2
		atyp_new = 0x04
	}
	fmt.Println("DEBUG || Local udp atyp:", atyp_new)

	// Send local address to client.

	_, err = client_tcp.Write(append([]byte{0x05, 0x00, 0x00, atyp_new}, local_addr...))
	if err != nil {
		client_udp.Close()
		remote_udp.Close()
		return nil
	}

	// Tag of whether tcp link is disconnected.
	ctx := context.Background()
	my_ctx, cancel := context.WithCancel(ctx)

	defer client_tcp.Close()
	defer client_udp.Close()
	defer remote_udp.Close()

	// Listen data.
	go Listen_Client_UDP(client_udp, remote_udp, client_addr)
	go Listen_Remote_UDP(client_udp, remote_udp, client_addr)
	go func() {
		for { // Check whether tcp is still linking
			_, err = client_tcp.Read(buf[:])
			if err != nil {
				break
			}
		}
		cancel()
	}()

	fmt.Println("Waiting......")

	// Checking the tag.
	select {
	case <-my_ctx.Done():
	}
	fmt.Println("end of this udp!")
	return nil
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

func Parse_IP_Port(str string) []byte {
	IP_str, port_str, _ := net.SplitHostPort(str)
	IP := net.ParseIP(IP_str)
	port, _ := strconv.Atoi(port_str)
	return binary.BigEndian.AppendUint16(IP, uint16(port))
}

func Listen_Client_UDP(client, remote *net.UDPConn, client_addr *net.UDPAddr) {
	defer client.Close()
	defer remote.Close()
	var buf [512]byte

	for {
		n, addr, err := client.ReadFromUDP(buf[:])
		fmt.Println("DEBUG || Listen Client")
		if err != nil {
			break
		}

		// Check the address first
		if addr.IP.To16().String() != client_addr.IP.To16().String() ||
			(client_addr.Port != 0 &&
				uint16(client_addr.Port) != uint16(addr.Port)) {
			continue
		}

		// Check the headings
		rsv1, rsv2, frag, atyp := int(buf[0]), int(buf[1]), int(buf[2]), int(buf[3])
		if rsv1 != 0x00 || rsv2 != 0x00 {
			fmt.Println("udp rsv not 0x00000000")
			continue
		}
		if frag != 0x00 {
			fmt.Println("udp don't support more frag than 0")
			continue
		}
		if atyp != 0x01 && atyp != 0x03 && atyp != 0x04 {
			fmt.Println("udp don't support this address type")
			continue
		}

		index := 0
		if atyp == 0x01 {
			index = 4 + 4 + 2
		} else if atyp == 0x04 {
			index = 4 + 16 + 2
		} else {
			index = 4 + int(buf[4]) + 2
		}

		// Get the remote address
		remote_addr, err := net.ResolveUDPAddr("udp",
			TCP_Address_Parse(buf[4:], atyp))
		if err != nil {
			continue
		}
		remote.WriteToUDP(buf[index:n], remote_addr)
	}
}

func Listen_Remote_UDP(client, remote *net.UDPConn, client_addr *net.UDPAddr) {
	defer client.Close()
	defer remote.Close()
	var buf [512]byte

	for {
		n, _, err := remote.ReadFromUDP(buf[:])
		fmt.Println("DEBUG || Listen Remote")
		if err != nil {
			return
		}

		// IP := addr.IP
		// port := uint16(addr.Port)
		rvl :=
			[]byte{}
		// []byte{0x00, 0x00, 0x00}

		// if IP.To4() != nil { // ipv4 version
		// 	rvl = append(rvl, 0x01)
		// 	rvl = append(rvl, binary.BigEndian.AppendUint16(IP.To4(), port)...)
		// } else { // ipv6 version
		// 	rvl = append(rvl, 0x04)
		// 	rvl = append(rvl, binary.BigEndian.AppendUint16(IP.To16(), port)...)
		// }
		client.WriteToUDP(append(rvl, buf[0:n]...), client_addr)
	}
}
