package main

import (
	"context"
	"fmt"
	"net"
)

func UDP_Connection(client_tcp net.Conn, atyp int, addr string) error {
	var buf [512]byte
	client_addr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		if err.Error() == "invalid address!" {
			client_tcp.Write([]byte{0x05, 0x08, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
		return err
	}
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

func Listen_Client_UDP(client, remote *net.UDPConn, client_addr *net.UDPAddr) {
	defer client.Close()
	defer remote.Close()
	buf := make([]byte, 32*1024)

	for {
		n, addr, err := client.ReadFromUDP(buf)
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
	buf := make([]byte, 32*1024)

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
