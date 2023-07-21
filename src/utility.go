package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Forward from target to client
func Forward_Target(client, target net.Conn) {
	defer target.Close()
	defer client.Close()

	const size = 32 * 1024
	tag := -1
	buf := make([]byte, size) // 32k buffer
	data := make([]byte, 0)
	lens := 0
	index := -1
	clens := int64(-1)
	compress := ""
	for {
		nr, er := target.Read(buf[:size])

		// Init case
		if tag == -1 {
			tag = Is_Response_Content(buf, nr)
		}
		// HTTP Response case
		if tag > 0 {
			// Main part.
			lens += nr
			data = append(data, buf[:nr]...)
			if index == -1 {
				// Tries to find the first index of the header
				index = strings.Index(string(data), "\r\n\r\n")
				if index != -1 {
					clens, compress = Http_Response_Parse(data, lens)
					if clens == -2 {
						fmt.Println("DEBUG || HTTP Parse error")
						return // HTTP Parse Error.
					}
				}
			}
			if index != -1 {
				if clens == -1 {
					if strings.HasSuffix(string(data[lens-5:lens]), "0\r\n\r\n") {
						data, lens = Http_Response_Decode(data, lens)
						if lens == -1 {
							fmt.Println("DEBUG || HTTP Parse error")
							return // HTTP Parse Error
						}
						tag = -1
					}
				} else {
					if lens == index+int(clens)+4 {
						tag = -1
					}
				}
			}

			if tag == -1 {
				data, lens = Http_Response_Modify(data, lens, compress)
				nw, ew := client.Write(data[:lens])
				if nw < lens || ew != nil {
					fmt.Println("DEBUG || End of forward! 3")
					return
				}
				data = data[:0]
				lens = 0
				index = -1
				clens = -1
			}
		} else {
			// No need to send 0 pack
			if nr > 0 {
				nw, ew := client.Write(buf[0:nr])
				if nw < nr || ew != nil {
					fmt.Println("DEBUG || End of forward! 1")
					return
				}
			}
		}

		// Deal with error and return!
		if er != nil {
			fmt.Println("DEBUG || End of forward! 2")
			return
		}
	}
}

// Forward from client to target
func Forward_Client(client, target net.Conn, str []byte, n int) {
	defer target.Close()
	defer client.Close()

	const size = 32 * 1024
	tag := -1
	buf := make([]byte, 0)
	if str != nil {
		buf = str
	} else {
		buf = make([]byte, size) // 32k buffer
	}
	data := make([]byte, 0)
	lens := 0

	for {
		nr := 0
		er := error(nil)
		if str != nil {
			str = nil // Now str goes useless......
			nr = n
		} else {
			nr, er = client.Read(buf[:size])
		}

		// Init case
		if tag == -1 {
			tag = Is_Request_Content(buf, nr)
		}
		// HTTP Request case
		if tag > 0 {
			lens += nr
			data = append(data, buf[:nr]...)
			headers := strings.Split(string(data), "\r\n\r\n")
			// End of GET case
			if headers[len(headers)-1] == "" {
				tag = -1        // Reset
				data = data[:0] // Reset
				lens = 0        // Reset
			}
		}

		// No need to send 0 pack
		if nr > 0 {
			nw, ew := target.Write(buf[0:nr])
			if nw < nr || ew != nil {
				fmt.Println("DEBUG || End of forward! 4")
				return
			}
		}

		// Deal with error!
		if er != nil {
			fmt.Println("DEBUG || End of forward! 5")
			return
		}
	}
}

func Forward_TCP(client, target net.Conn, reader *bufio.Reader) {
	go Forward_Target(client, target)
	if reader == nil {
		go Forward_Client(client, target, nil, 114514)
	} else {
		const size = 32 * 1024
		buf := make([]byte, size)
		nr, er := reader.Read(buf[:size])
		if er != nil {
			fmt.Println("DEBUG || Error :", er)
			client.Close()
			target.Close()
			return
		}
		go Forward_Client(client, target, buf, nr)
	}
}

// Parse IP and port.
func Parse_IP_Port(str string) []byte {
	ip_str, port_str, _ := net.SplitHostPort(str)
	ip := net.ParseIP(ip_str)
	port, _ := strconv.Atoi(port_str)
	return binary.BigEndian.AppendUint16(ip, uint16(port))
}

// Ipv4 : len = 4  + 2 = 6
// Ipv6 : len = 16 + 2 = 18
func Get_Atyp(IP []byte) int {
	if len(IP) == 6 { // ipv4 = 4 + 2
		return 0x01
	} else { // ipv6 = 16 + 2
		return 0x04
	}
}

// Return whether it is TLS link
func Is_TLS(str []byte) bool {
	return str[0] == 0x16 && str[1] == 0x03 && str[2] == 0x01
}

// Return   0 if not HTTP
// Return > 0 if HTTP Method
func Is_Request_Content(buf []byte, n int) int {
	if n > 8 {
		n = 8
	}
	str := string(buf[:n])
	switch {
	case strings.HasPrefix(str, "GET"):
		return 1
	case strings.HasPrefix(str, "POST"):
		return 2
	case strings.HasPrefix(str, "PUT"):
		return 3
	case strings.HasPrefix(str, "DELETE"):
		return 4
	case strings.HasPrefix(str, "PATCH"):
		return 5
	case strings.HasPrefix(str, "HEAD"):
		return 6
	case strings.HasPrefix(str, "OPTIONS"):
		return 7
	default:
		return 0
	}
}

func Is_Response_Content(buf []byte, n int) int {
	if n > 8 {
		n = 8
	}
	if strings.HasPrefix(string(buf[:n]), "HTTP") {
		return 1
	} else {
		return 0
	}
}
