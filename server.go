package main

import (
	"fmt"
	"net"
	"os"
)

type RespGen func(*Req) string

// Path -> method -> generator
var paths = make(map[string]map[string]RespGen)

func Get(path string, fn RespGen) {
	addRequest("GET", path, fn)
}

func Post(path string, fn RespGen) {
	addRequest("POST", path, fn)
}

func addRequest(method string, path string, fn RespGen) {
	if _, ok := paths[path]; !ok {
		paths[path] = make(map[string]RespGen)
	}
	paths[path][method] = fn
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Accepted connection from", conn.RemoteAddr())

	buff := make([]byte, 1024)

	// Request before switching protocols
	var http1Req *Req

	for {
		n, err := conn.Read(buff)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Printf("Error reading from %s: %s\n", conn.RemoteAddr(), err.Error())
			}
			break
		}

		if http1Req == nil {
			http1Req, err = handleHttp1(conn, buff[:n])
			if err != nil {
				fmt.Printf("Error handling HTTP/1 request for %s: %s\n", conn.RemoteAddr(), err.Error())
				break
			}

			continue
		}

		err = handleHttp2(conn, buff[:n], http1Req)
		if err != nil {
			fmt.Printf("Error handling HTTP/2 request for %s: %s\n", conn.RemoteAddr(), err.Error())
			break
		}
	}

	fmt.Println("Connection was closed for", conn.RemoteAddr())
}

func writeConn(conn net.Conn, msg []byte) error {
	_, err := conn.Write(msg)
	return err
}

func Start() {
	listener, err := net.Listen("tcp", ":80")
	if err != nil {
		fmt.Println("Error listening on port 80:", err.Error())
		os.Exit(1)
	}

	fmt.Println("Waiting for connections on port 80")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection on port 80:", err.Error())
			continue
		}

		go handleConn(conn)
	}
}
