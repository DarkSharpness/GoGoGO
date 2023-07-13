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
// Return 2 if HTTP NOT GET
func Is_Http_Content(buf []byte, n int) int {
	if n > 16 {
		n = 16
	}
	str := string(buf[:n])
	if strings.Contains(str, "HTTP") {
		if strings.Contains(str, "GET") {
			return 1
		} else {
			return 2
		}
	} else {
		return 0
	}
}

func Http_Parse(buf []byte, n int) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(buf)), nil)
	if err != nil {
		fmt.Println("DEBUG || What the fuck is that:", err)
		return
	} else {
		fmt.Println("DEBUG || Pass the parse!")
	}
	cookie := resp.Cookies()
	lens := len(cookie)
	for i := 0; i < lens; i++ {
		fmt.Println("DEBUG ||", cookie[i].Value)
	}
	return
}
