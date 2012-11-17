package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"time"
)

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

type TcpProxy struct {
	localAddresses []string
	remoteAddress  string
	connectTimeout time.Duration
}

func NewTcpProxy(
	localAddresses []string, remoteAddress string,
	connectTimeout time.Duration) *TcpProxy {
	return &TcpProxy{
		localAddresses: localAddresses,
		remoteAddress:  remoteAddress,
		connectTimeout: connectTimeout}
}

func (proxy *TcpProxy) Start() {
	for _, localAddress := range proxy.localAddresses {
		go proxy.accept(localAddress)
	}
}

func (proxy *TcpProxy) accept(localAddr string) {
	local, err := net.Listen("tcp", localAddr)
	logger.Printf("listening on %v", localAddr)
	if err != nil {
		logger.Fatal("cannot listen: ", err)
	}
	for {
		clientConnection, err := local.Accept()
		if err != nil {
			logger.Printf("accept failed: %v", err)
		} else {
			go proxy.handleClient(clientConnection)
		}
	}
}

func (proxy *TcpProxy) handleClient(clientConnection net.Conn) {
	clientConnectionString := buildClientConnectionString(clientConnection)
	logger.Printf("accept %v", clientConnectionString)

	remoteConnection, err := net.DialTimeout("tcp",
		proxy.remoteAddress, proxy.connectTimeout)
	if err != nil {
		logger.Printf("remote dial failed: %v", err)
		clientConnection.Close()
	} else {
		remoteConnectionString := buildRemoteConnectionString(remoteConnection)
		logger.Printf("connect %v", remoteConnectionString)
		go proxyConnections(
			clientConnection, remoteConnection, clientConnectionString)
		proxyConnections(
			remoteConnection, clientConnection, remoteConnectionString)
	}
}

func proxyConnections(
	source net.Conn, dest net.Conn, connectionString string) {
	defer source.Close()
	defer dest.Close()
	io.Copy(dest, source)
	logger.Printf("close %v", connectionString)
}

func buildClientConnectionString(clientConnection net.Conn) string {
	return fmt.Sprintf(
		"%v -> %v",
		clientConnection.RemoteAddr(),
		clientConnection.LocalAddr())
}

func buildRemoteConnectionString(remoteConnection net.Conn) string {
	return fmt.Sprintf(
		"%v -> %v",
		remoteConnection.LocalAddr(),
		remoteConnection.RemoteAddr())
}

func setNumProcs() {
	newMaxProcs := runtime.NumCPU()
	prevMaxProcs := runtime.GOMAXPROCS(newMaxProcs)
	logger.Printf(
		"set GOMAXPROCS = %v, prev GOMAXPROCS = %v",
		newMaxProcs, prevMaxProcs)
}

func main() {
	if len(os.Args) < 3 {
		logger.Fatal("usage: proxy <local> [ <local> ... ] <remote>")
	}

	setNumProcs()

	remoteAddress := os.Args[len(os.Args)-1]
	localAddresses := os.Args[1 : len(os.Args)-1]

	proxy := NewTcpProxy(localAddresses, remoteAddress, 10*time.Second)
	proxy.Start()

	for {
		time.Sleep(10 * time.Second)
	}
}
