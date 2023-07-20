package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"time"
)

func generateCert(host string) (*tls.Certificate, error) {
	host, _, _ = net.SplitHostPort(host)

	// 读取根证书
	rootCertPEM, err := ioutil.ReadFile("root.crt") // 改变文件名
	if err != nil {
		return nil,
			errors.New(fmt.Sprintf("Failed to read root certificate: %v", err))
	}
	block, _ := pem.Decode(rootCertPEM)
	if block == nil {
		return nil,
			errors.New("Failed to decode PEM block containing the certificate")
	}
	rootCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil,
			errors.New(fmt.Sprintf("Failed to parse certificate: %v", err))
	}

	// 读取私钥
	rootKeyPEM, err := ioutil.ReadFile("decrypted.key") // 改变文件名
	if err != nil {
		return nil,
			errors.New(fmt.Sprintf("Failed to read private key: %v", err))
	}
	block, _ = pem.Decode(rootKeyPEM)
	if block == nil {
		return nil,
			errors.New("Failed to decode PEM block containing the private key")
	}
	rootKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil,
			errors.New(fmt.Sprintf("Failed to parse private key: %v", err))
	}

	next, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(2), // 为了确保每个证书的序列号是唯一的，你可能需要动态生成这个值
		Subject: pkix.Name{
			CommonName:   host, // 这里设置你要签发的主机名
			Organization: []string{"My Company"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180), // 180 days
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{host}, // 这里设置你要签发的主机名
	}

	// 使用根证书签发新的证书
	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, rootCert, &next.PublicKey, rootKey)

	// 将新的证书和私钥编码为 PEM 格式
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes, _ := x509.MarshalECPrivateKey(next)
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	// Now you can use certPem and keyPem for tls.X509KeyPair()

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil,
			errors.New(fmt.Sprintf("Creating certificate: %s", err))
	}
	return &cert, nil
}

func TLS_Build_Proxy(target_addr string) (string, error) {
	// A new local server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	fmt.Println("Fuck you! TLS!")
	go func() {
		// Function that current node is doing.
		server_tcp, er := listener.Accept()
		if er != nil {
			fmt.Println("Handshake error!", er)
			listener.Close()
			return
		}

		// Generate certification.
		cert, er := generateCert(target_addr)
		if er != nil {
			fmt.Println("Fail to generate fake certificate!", er)
			listener.Close()
			server_tcp.Close()
			return
		}
		config := &tls.Config{
			Certificates:       []tls.Certificate{*cert},
			InsecureSkipVerify: true,
		}

		server_tls := tls.Server(server_tcp, config)
		er = server_tls.Handshake()
		if er != nil {
			listener.Close()
			server_tls.Close()
			fmt.Println(er.Error())
			return
		}
		fmt.Println("DEBUG || Receive TLS handshake")

		config = &tls.Config{InsecureSkipVerify: true}
		target, err := tls.Dial("tcp", target_addr, config)
		if err != nil {
			listener.Close()
			server_tls.Close()
			fmt.Println(err.Error())
			return
		}
		go Forward_TCP(server_tls, target, nil)
	}()
	return listener.Addr().String(), nil
}
