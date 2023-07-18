package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
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

	for {
		nr, er := target.Read(buf[:size])

		// Init case
		if tag == -1 {
			tag = Is_Response_Content(buf, nr)
			fmt.Println("DEBUG || RESPONSE TAG")
		}
		// HTTP Response case
		if tag > 0 {
			// Debug only
			file, _ := os.OpenFile("/home/GoGoGo/Ignore/response.html",
				os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			file.Write(buf[0:nr])
			file.WriteString("\n-------------------------------------------------\n")

			// Main part.
			lens += nr
			data = append(data, buf[:nr]...)
			if index == -1 {
				// Tries to find the first index of the header
				index = strings.Index(string(data), "\r\n\r\n")
				if index != -1 {
					clens = Http_Response_Parse(data, lens)
					if clens == -2 {
						fmt.Println("DEBUG || ?????")
						return // HTTP Parse Error.
					}
				}
			}
			if index != -1 {
				if clens == -1 {
					if strings.HasSuffix(string(data[lens-5:lens]), "0\r\n\r\n") {
						data, lens = Http_Response_Decode(data, lens)
						if lens == -1 {
							fmt.Println("DEBUG || ???")
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
				fmt.Println("DEBUG || Get complete message qwq!")
				Http_Request_Modify(data, lens)
				nw, ew := client.Write(data[:lens])
				if nw < lens || ew != nil {
					fmt.Println("DEBUG || End of forward operation 3!")
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
					fmt.Println("DEBUG || End of forward operation 3!")
					return
				}
			}
		}

		// Deal with error and return!
		if er != nil {
			fmt.Println("DEBUG || End of forward operation 4!", er)
			return
		}
	}
}

// Forward from client to target
func Forward_Client(client, target net.Conn) {
	defer client.Close()
	defer target.Close()

	const size = 32 * 1024
	tag := -1
	buf := make([]byte, size) // 32k buffer
	data := make([]byte, 0)
	lens := 0

	for {
		nr, er := client.Read(buf[:size])

		// Init case
		if tag == -1 {
			tag = Is_Request_Content(buf, nr)
			fmt.Println("DEBUG || TAG", tag)
		}
		// HTTP Request case
		if tag > 0 {
			// Debug only
			file, _ := os.OpenFile("/home/GoGoGo/Ignore/request.html",
				os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			file.Write(buf[0:nr])
			file.WriteString("\n-------------------------------------------------\n")

			lens += nr
			data = append(data, buf[:nr]...)
			headers := strings.Split(string(data), "\r\n\r\n")
			// End of GET case
			if headers[len(headers)-1] == "" {
				fmt.Println("DEBUG || True end of forward!")
				tag = -1        // Reset
				data = data[:0] // Reset
				lens = 0        // Reset
			}
		}

		// No need to send 0 pack
		if nr > 0 {
			nw, ew := target.Write(buf[0:nr])
			if nw < nr || ew != nil {
				fmt.Println("DEBUG || End of forward operation 1!", ew)
				return
			}
		}

		// Deal with error!
		if er != nil {
			fmt.Println("DEBUG || End of forward operation 2!", er)
			return
		}
	}
}

// Parse IP and port.
func Parse_IP_Port(str string) []byte {
	ip_str, port_str, _ := net.SplitHostPort(str)
	ip := net.ParseIP(ip_str)
	port, _ := strconv.Atoi(port_str)
	return binary.BigEndian.AppendUint16(ip, uint16(port))
}
