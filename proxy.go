package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"time"
)

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

func main() {
	if len(os.Args) < 3 {
		logger.Fatal("usage: proxy <local> [ <local> ... ] <remote>")
	}

	setNumProcs()

	remoteAddr := os.Args[len(os.Args)-1]
	localAddrs := os.Args[1 : len(os.Args)-1]
	for _, localAddr := range localAddrs {
		go accept(localAddr, remoteAddr)
	}

	for {
		time.Sleep(10 * time.Second)
	}
}

func setNumProcs() {
	numCPU := runtime.NumCPU()
	prevMaxProcs := runtime.GOMAXPROCS(numCPU)
	logger.Printf(
		"set GOMAXPROCS = NumCPU = %v, prev GOMAXPROCS = %v",
		numCPU, prevMaxProcs)
}

func accept(localAddr string, remoteAddr string) {
	local, err := net.Listen("tcp", localAddr)
	logger.Printf("listening on %v\n", localAddr)
	if err != nil {
		logger.Fatal("cannot listen: %v", err)
	}
	for {
		clientConnection, err := local.Accept()
		if err != nil {
			logger.Fatal("accept failed: %v", err)
		}
		go handleClient(clientConnection, remoteAddr)
	}
}

func handleClient(clientConnection net.Conn, remoteAddr string) {
	clientConnectionString := buildClientConnectionString(clientConnection)
	logger.Printf("accepted %v\n", clientConnectionString)

	remoteConnection, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		logger.Printf("remote dial failed: %v\n", err)
		clientConnection.Close()
	} else {
		go proxyConnections(
			clientConnection, remoteConnection, clientConnectionString)
		remoteConnectionString := buildRemoteConnectionString(remoteConnection)
		proxyConnections(
			remoteConnection, clientConnection, remoteConnectionString)
	}
}

func proxyConnections(
	connection1 net.Conn, connection2 net.Conn, connectionString string) {
	io.Copy(connection1, connection2)
	connection1.Close()
	connection2.Close()
	logger.Printf("closed %v\n", connectionString)
}

func buildClientConnectionString(clientConnection net.Conn) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(
		buf, "%v -> %v",
		clientConnection.RemoteAddr(), clientConnection.LocalAddr())
	return buf.String()
}

func buildRemoteConnectionString(remoteConnection net.Conn) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(
		buf, "%v -> %v",
		remoteConnection.LocalAddr(), remoteConnection.RemoteAddr())
	return buf.String()
}
