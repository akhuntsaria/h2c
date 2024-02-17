package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func handleConn(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Accepted connection from", conn.RemoteAddr())

	buff := make([]byte, 1024)
	for {
		n, err := conn.Read(buff)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Printf("Error reading from %s: %s\n", conn.RemoteAddr(), err.Error())
			}
			break
		}

		msg := string(buff[:n])
		fmt.Printf("Received %d bytes from %s: %s", n, conn.RemoteAddr(), msg)

		responded := false

		lines := strings.Split(strings.TrimSpace(msg), "\n")
		if len(lines) != 0 {
			firstLine := strings.TrimSpace(lines[0])
			if firstLine == "GET /ping HTTP/1.1" {
				response := "pong"
				msg = fmt.Sprintf("HTTP/1.1 200 OK\nContent-Length: %d\n\n%s", len(response), response)

				if !writeConn(conn, msg) {
					break
				} else {
					responded = true
				}
			}
		}

		if !responded {
			msg = "HTTP/1.1 400 Bad Request\nContent-Length: 0\n\n"
			if !writeConn(conn, msg) {
				break
			}
		}
	}

	fmt.Println("Closed connection from", conn.RemoteAddr())
}

func writeConn(conn net.Conn, msg string) bool {
	fmt.Printf("Writing to %s: %s\n", conn.RemoteAddr(), msg)

	_, err := conn.Write([]byte(msg))
	if err != nil {
		fmt.Printf("Error writing to %s: %s\n", conn.RemoteAddr(), err.Error())
		return false
	}

	return true
}

func main() {
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
