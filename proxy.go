package main

import (
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
	logger.Print("listening on ", localAddr)
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
	logger.Print("accept ", clientConnectionString)

	remoteConnection, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		logger.Printf("remote dial failed: %v", err)
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
	source net.Conn, dest net.Conn, connectionString string) {
	defer source.Close()
	defer dest.Close()
	io.Copy(dest, source)
	logger.Print("close ", connectionString)
}

func buildClientConnectionString(clientConnection net.Conn) string {
	return (clientConnection.RemoteAddr().String() + " -> " +
		clientConnection.LocalAddr().String())
}

func buildRemoteConnectionString(remoteConnection net.Conn) string {
	return (remoteConnection.LocalAddr().String() + " -> " +
		remoteConnection.RemoteAddr().String())
}
