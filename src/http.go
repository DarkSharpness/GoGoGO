package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

// Return 0 if not HTTP
// Return 1 if HTTP GET
// Return 2 if HTTP POST
// Return 3
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

// Parse HTTP GET
func Http_Request_Parse(buf []byte, n int) error {
	fmt.Println("DEBUG ||", string(buf[:n]))
	_, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buf)))
	return err
}

// Parse HTTP GET
// Return the content length
func Http_Response_Parse(buf []byte, n int) int64 {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(buf)), nil)
	if err != nil {
		return -2
	}
	return resp.ContentLength
}

// Decode for Transfer-Encoding: chunked
// Return a new array of data.
func Http_Response_Decode(buf []byte, n int) ([]byte, int) {
	str := string(buf)
	data := make([]byte, 0)
	head := make([]byte, 0)
	index := strings.Index(str, "\r\n\r\n") + 4
	head = append(head, str[:index]...)
	fmt.Printf("Origin:\n%v\n", string(buf))
	for {
		size := 0
		fmt.Sscanf(str[index:], "%x", &size)
		if size == 0 {
			break
		}
		index += strings.Index(str[index:], "\r\n") + 2
		data = append(data, str[index:index+size]...)
		index += size + 2
	}

	// Here, do something to data.
	data = bytes.Replace(data, []byte("a"), []byte("bb"), -1)

	head = append(head, "\r\n"...)
	head = append(head, []byte(fmt.Sprintf("%x", len(data)))...)
	head = append(head, "\r\n"...)
	head = append(head, data...)
	head = append(head, "\r\n0\r\n\r\n"...)
	fmt.Printf("Final:\n%v\n", string(head))
	return data, 0
}

func Http_Request_Modify(buf []byte, n int) {
}
