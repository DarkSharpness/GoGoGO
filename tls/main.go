package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

func main() {
	cert, err := tls.LoadX509KeyPair("localhost.crt", "localhost.key")

	if err != nil {
		fmt.Printf("proxy: loadkeys: %s", err)
		return
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	proxy := &http.Server{
		Addr:      "127.0.0.1:8080",
		Handler:   http.HandlerFunc(HandleRequest),
		TLSConfig: config,
	}

	go func() {
		proxy.ListenAndServeTLS("localhost.crt", "localhost.key")
	}()

	fmt.Println("Genshin Impact, LAUNCH!")
	select {}
}

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		fmt.Println("CONNECT Method!")
		HandleConnect(w, r)
	} else {
		fmt.Println("HTTP Method!")
		HandleHttp(w, r)
	}
}
