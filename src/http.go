package main

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
)

// Parse HTTP GET
func Http_Request_Parse(buf []byte, n int) error {
	_, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buf)))
	return err
}

// Parse HTTP GET
// Return the content length
func Http_Response_Parse(buf []byte, n int) (int64, string) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(buf)), nil)
	if err != nil {
		return -2, ""
	}
	return resp.ContentLength, resp.Header.Get("Content-Encoding")
}

// Decode for Transfer-Encoding: chunked
// Return a new array of data.
func Http_Response_Decode(buf []byte, n int) ([]byte, int) {
	str := string(buf)

	// Index
	index := strings.Index(str, "\r\n\r\n") + 4
	data := make([]byte, 0)
	head := buf[:index]

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
	return head, 0
}

// Decode for Fix content-length.
func Http_Response_Modify(buf []byte, n int, compress string) ([]byte, int) {
	index := bytes.Index(buf, []byte{'\r', '\n', '\r', '\n'}) + 4

	// Here, do something to data.
	data := Http_Body_Modify(buf[index:], compress)
	head := Http_Head_Update(buf[:index], len(data))

	head = append(head, data...)
	return head, len(head)
}

// Modify head information
func Http_Head_Update(buf []byte, n int) []byte {
	slice := strings.Split(string(buf), "\r\n")
	target := "Content-Length: " + fmt.Sprintf("%d", n)
	j := -1
	for i := 0; i < len(slice); i++ {
		if strings.HasPrefix(slice[i], "Content-Length:") {
			fmt.Println("Code:", slice[i])
			slice[i] = target
		}
		if strings.HasPrefix(slice[i], "Transfer-Encoding: chunked") {
			fmt.Println("Code:", slice[i])
			slice[i] = target
		}
		if strings.HasPrefix(slice[i], "Content-Encoding") {
			switch slice[i][18:] {
			case "gzip":
			case "br":
			case "deflate":
			case "": // Just break!!!
			default: // Will not modify unknown encoding.
				continue
			}
			j = i
		}
	}
	if j != -1 {
		slice = append(slice[:j], slice[j+1:]...)
	}
	return []byte(strings.Join(slice, "\r\n"))
}

// Decompress and modify the data.
func Http_Body_Modify(data []byte, compress string) []byte {
	fmt.Println("Compress:", compress)
	switch compress {
	case "gzip":
		buffer := bytes.Buffer{}
		gzipReader, _ := gzip.NewReader(bytes.NewReader(data))
		if gzipReader == nil {
			break
		}
		io.Copy(&buffer, gzipReader)
		gzipReader.Close()
		data = buffer.Bytes()
	case "br":
		brReader := brotli.NewReader(bytes.NewReader(data))
		data, _ = ioutil.ReadAll(brReader)
	case "deflate":
		flateReader := flate.NewReader(bytes.NewReader(data))
		data, _ = ioutil.ReadAll(flateReader)
	case "": // No need to decompress
	default: // Will not modify unknown encoding
		return data
	}
	return bytes.Replace(data, []byte("百度"), []byte("DarkSharpness"), -1)
}
