package tls

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
)

func HandleConnect(w http.ResponseWriter, r *http.Request) {
	hijacker, state := w.(http.Hijacker)
	if !state {
		// Error code sending
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	client, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Loading certificate
	cert, err := tls.LoadX509KeyPair("localhost.crt", "localhost.key")
	if err != nil {
		fmt.Printf("proxy: loadkeys: %s", err)
		return
	}

	// Send ok message
	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Config for tls
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	server := tls.Server(client, config)

	remote, err := tls.Dial("tcp", r.Host, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		// Error code sending
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	Connect(server, remote)
}

// Connect target and client
func Connect(client, target net.Conn) {
	fmt.Println("Connection Success!")
	// Forward 2 connection
	go Forward_Client(client, target)
	go Forward_Target(target, client)
}

// Forward from target to client
func Forward_Target(target, client net.Conn) {
	defer target.Close()
	defer client.Close()
	io.Copy(client, target)
}

// Custom io.Copy function awa.
func Forward_Client(client, target net.Conn) {
	defer client.Close()
	defer target.Close()

	const size = 32 * 1024
	buf := make([]byte, size) // 32k buffer

	for {
		nr, er := client.Read(buf[:size])

		// Do something here qwq.

		// No need to send 0 pack
		if nr > 0 {
			nw, ew := target.Write(buf[:nr])
			fmt.Println("DEBUG ||", string(buf[:nr]))
			if nw < nr || ew != nil {
				fmt.Println("DEBUG || End of forward operation!")
				return
			}
		}

		// Deal with error!
		if er != nil {
			fmt.Println("DEBUG || End of forward operation!")
			return
		}
	}
}

func HandleHttp(w http.ResponseWriter, r *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Copy the headers. (Fuck golang!)
	CopyHeader(w.Header(), resp.Header)
	// Write status code.
	w.WriteHeader(resp.StatusCode)

	// Copy data within
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}

// Fxxk golang!
func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
