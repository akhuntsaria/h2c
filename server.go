package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/http2/hpack"
)

// HTTP/2 frame
type Frame struct {
	frameType, flags byte
	streamId         int
	payload          []byte
}

type RespGen func() string

var (
	CONN_PREFACE        = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
	FRAME_TYPE_SETTINGS = 0x4
)

var paths = make(map[string]RespGen)

func Get(path string, fn RespGen) {
	paths[path] = fn
}

func bytesStartWith(src []byte, search []byte) bool {
	if len(src) < len(search) {
		return false
	}

	for i := range search {
		if src[i] != search[i] {
			return false
		}
	}

	return true
}

// [0x00, 0xff, 0x01] -> 65281
func bytesToDec(bin []byte) int {
	dec := 0
	for i := 0; i < len(bin); i++ {
		dec |= int(bin[i]) << (8 * (len(bin) - i - 1))
	}
	return dec
}

// 65281, 2 -> [0xff, 0x01]
func decToBytes(dec int, len int) []byte {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(dec >> (8 * (len - i - 1)))
	}
	return bytes
}

func frameToBytes(frame Frame) []byte {
	bytes := []byte{}

	payloadLen := decToBytes(len(frame.payload), 3)
	bytes = append(bytes, payloadLen...)

	bytes = append(bytes, frame.frameType, frame.flags)

	streamId := decToBytes(frame.streamId, 4)
	bytes = append(bytes, streamId...)

	bytes = append(bytes, frame.payload...)
	return bytes
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Accepted connection from", conn.RemoteAddr())

	buff := make([]byte, 1024)
	http2 := false
	http2SettingsAcked := false
	http2DataSend := false
	var http2Path string
	for {
		n, err := conn.Read(buff)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Printf("Error reading from %s: %s\n", conn.RemoteAddr(), err.Error())
			}
			break
		}

		if http2 {
			frames, err := bytesToFrames(buff[:n])
			if err != nil {
				fmt.Printf("Error parsing HTTP/2 packet from %s: %s\n", conn.RemoteAddr(), err.Error())
				break
			}

			fmt.Printf("HTTP/2 frames recieved from %s: %s\n", conn.RemoteAddr(), fmt.Sprint(frames))

			msg := []byte{}
			for _, frame := range frames {
				// Nothing to process, just mirror the settings
				msg = append(msg, frameToBytes(frame)...)

				if isSettingsFrame(frame) && isAckFlagSet(frame) {
					http2SettingsAcked = true
				}
			}

			fmt.Println("Exchanging HTTP/2 settings with", conn.RemoteAddr())
			writeConn(conn, msg)

			if !http2SettingsAcked || http2DataSend {
				continue
			}

			fmt.Println("Sending HTTP/2 headers and data to", conn.RemoteAddr())

			notFound := false

			if fn, ok := paths[http2Path]; ok {
				msg = []byte(fn())
			} else {
				notFound = true
				msg = []byte{}
			}

			var headersBuff bytes.Buffer
			encoder := hpack.NewEncoder(&headersBuff)
			var status string
			if notFound {
				status = "404"
			} else {
				status = "200"
			}
			encoder.WriteField(hpack.HeaderField{Name: ":status", Value: status, Sensitive: false})
			encoder.WriteField(hpack.HeaderField{Name: "content-length", Value: strconv.Itoa(len(msg)), Sensitive: false})
			headersPayload := headersBuff.Bytes()

			headers := Frame{
				0x1,
				0x4, // END_HEADERS
				1,
				headersPayload,
			}

			data := Frame{
				0x0,
				0x1, // END_STREAM,
				1,
				msg,
			}

			headersAndData := append(frameToBytes(headers), frameToBytes(data)...)
			writeConn(conn, headersAndData)
			http2DataSend = true
			continue
		}

		msg := string(buff[:n])
		// Leave only headers
		msg = msg[:strings.Index(msg, "\r\n\r\n")]

		lines := strings.Split(strings.TrimSpace(msg), "\r\n")
		responded := false
		for _, a := range lines {
			if a == "Upgrade: h2c" {
				fmt.Println("Switching to HTTP/2 for", conn.RemoteAddr())

				msg := []byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: h2c\r\nConnection: Upgrade\r\n\r\n")
				if !writeConn(conn, msg) {
					break
				}

				http2 = true
				http2Path = strings.Split(strings.TrimSpace(lines[0]), " ")[1]
				responded = true
				break
			}
		}

		if responded {
			continue
		}

		if len(lines) != 0 {
			firstLine := strings.TrimSpace(lines[0])
			parts := strings.Split(firstLine, " ")
			path := parts[1]
			if fn, ok := paths[path]; ok {
				response := fn()
				msg := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", len(response), response))

				if !writeConn(conn, msg) {
					break
				} else {
					responded = true
				}
			}
		}

		if !responded {
			msg := []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n")
			if !writeConn(conn, msg) {
				break
			}
		}
	}

	fmt.Println("Connection was closed for", conn.RemoteAddr())
}

func isAckFlagSet(frame Frame) bool {
	return frame.flags == 0x1
}

func isSettingsFrame(frame Frame) bool {
	return frame.frameType == 0x4
}

func bytesToFrames(buff []byte) ([]Frame, error) {
	if len(buff) == 0 {
		return nil, errors.New("empty input")
	}

	cur := 0

	if bytesStartWith(buff, CONN_PREFACE) {
		// Ignore the preface, set cursor after it
		cur = len(CONN_PREFACE)
	}

	frames := []Frame{}
	for cur < len(buff) {
		frame := Frame{}
		payloadSize := bytesToDec(buff[cur : cur+3])
		cur += 3

		frame.frameType = buff[cur]
		cur++

		frame.flags = buff[cur]
		cur++

		frame.streamId = bytesToDec(buff[cur : cur+4])
		cur += 4

		frame.payload = buff[cur : cur+payloadSize]
		cur += payloadSize

		frames = append(frames, frame)
	}
	return frames, nil
}

func writeConn(conn net.Conn, msg []byte) bool {
	_, err := conn.Write(msg)
	if err != nil {
		fmt.Printf("Error writing to %s: %s\n", conn.RemoteAddr(), err.Error())
		return false
	}

	return true
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
