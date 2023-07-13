package main

import "strings"

func Is_Http_Content(buf []byte, n int) {
	if n > 64 {
		n = 64
	}
	if strings.Contains(string(buf[:n]),"H") {
	}
}
