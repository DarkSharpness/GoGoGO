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
	fmt.Printf("Origin:\n%v\n", str)

	// Index
	index := strings.Index(str, "\r\n\r\n") + 4
	data := make([]byte, 0)
	head := make([]byte, 0)
	head = append(head, str[:index]...)
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

	head = append(head, data...)
	fmt.Printf("Final:\n%v\n", string(head))
	return head, 0
}

// Decode for Fix content-length.
func Http_Response_Modify(buf []byte, n int) ([]byte, int) {
	index := strings.Index(string(buf), "\r\n\r\n")
	fmt.Printf("Origin2:\n%v\n", string(buf))
	data := make([]byte, 0)
	data = append(data, buf[index:]...)

	// Here, do something to data.
	// data = bytes.Replace(data, []byte("努力"), []byte("天天摆烂"), -1)

	head := Http_Response_Modify_Head(string(buf[:index]), len(data) - 4)

	head = append(head, data...)
	fmt.Printf("Final2:\n%v\n", string(head))
	fmt.Println("Compare",len(head) - len(buf))

	return head, len(head)
}

// Modify head information
func Http_Response_Modify_Head(str string, n int) []byte {
	slice := strings.Split(str, "\r\n")
	target := "Content-Length: " + fmt.Sprintf("%d", n)
	for i := 0; i < len(slice); i++ {
		if strings.HasPrefix(slice[i], "Content-Length:") {
			fmt.Println("Code:", slice[i])
			slice[i] = target
			break
		}
		if strings.HasPrefix(slice[i], "Transfer-Encoding: chunked") {
			fmt.Println("Code:", slice[i])
			slice[i] = target
			break
		}
	}
	return []byte(strings.Join(slice, "\r\n"))
}
