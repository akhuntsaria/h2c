package main

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Req struct {
	method, path string
	headers      map[string]string
	body         string
}

var (
	HEADERS_SEPARATOR = "\r\n\r\n"

	SWITCH_PROTOCOL_RESPONSE = []byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: h2c\r\nConnection: Upgrade\r\n\r\n")
)

func bytesToRequest(reqBytes []byte) *Req {
	reqStr := string(reqBytes)
	request := Req{}

	sepIdx := strings.Index(reqStr, HEADERS_SEPARATOR)
	headersStr := reqStr[:sepIdx]

	lines := strings.Split(strings.TrimSpace(headersStr), "\r\n")
	firstLineParts := strings.Split(lines[0], " ")

	request.method = firstLineParts[0]
	request.path = firstLineParts[1]
	request.headers = make(map[string]string)
	request.body = reqStr[sepIdx+len(HEADERS_SEPARATOR):]

	for i := 1; i < len(lines); i++ {
		headerParts := strings.Split(lines[i], ": ")
		request.headers[headerParts[0]] = headerParts[1]
	}
	return &request
}

func handleHttp1(conn net.Conn, buff []byte) (*Req, error) {
	req := bytesToRequest(buff)
	fmt.Printf("HTTP/1 request received from %s: %s\n", conn.RemoteAddr(), req)

	if !upgradeRequested(req) {
		return nil, errors.New("upgrade was not requested ")
	}
	fmt.Println("Switching to HTTP/2 for", conn.RemoteAddr())

	err := writeConn(conn, SWITCH_PROTOCOL_RESPONSE)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func upgradeRequested(req *Req) bool {
	return req.headers["Upgrade"] == "h2c"
}
