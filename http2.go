package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"

	"golang.org/x/net/http2/hpack"
)

type Frame struct {
	frameType, flags byte
	streamId         int
	payload          []byte
}

var (
	CONN_PREFACE = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")

	FLAG_ACK = byte(0x1)

	FRAME_TYPE_DATA          = byte(0x0)
	FRAME_TYPE_HEADERS       = byte(0x1)
	FRAME_TYPE_SETTINGS      = byte(0x4)
	FRAME_TYPE_WINDOW_UPDATE = byte(0x8)

	// One instance, to keep the reference table updated
	decoder = hpack.NewDecoder(4096, nil)
)

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

func handleHttp2(conn net.Conn, buff []byte, http1Req *Request) error {
	reqFrames, err := bytesToFrames(buff)
	if err != nil {
		return err
	}

	fmt.Printf("HTTP/2 frames recieved from %s: %s\n", conn.RemoteAddr(), fmt.Sprint(reqFrames))

	msg := []byte{}
	for _, reqFrame := range reqFrames {
		if reqFrame.frameType == FRAME_TYPE_SETTINGS && (reqFrame.flags&FLAG_ACK != 0) && http1Req != nil {
			// Process the original request, before switching protocols

			// ACKed settings frame has to be sent in a separate packet
			err = writeConn(conn, frameToBytes(reqFrame))
			if err != nil {
				return err
			}

			msg = append(msg, getHeadersAndData(conn, &http1Req.path, 1)...)
			http1Req = nil

		} else if reqFrame.frameType == FRAME_TYPE_HEADERS {
			// Process a new request
			path := getPath(reqFrame)
			if path != nil {
				msg = append(msg, getHeadersAndData(conn, path, reqFrame.streamId)...)
			}

		} else if reqFrame.frameType == FRAME_TYPE_SETTINGS || reqFrame.frameType == FRAME_TYPE_WINDOW_UPDATE {
			// Nothing to process, just mirror the settings
			msg = append(msg, frameToBytes(reqFrame)...)
		}
	}

	return writeConn(conn, msg)
}

func getHeadersAndData(conn net.Conn, path *string, streamId int) []byte {
	fmt.Println("Sending HTTP/2 headers and data to", conn.RemoteAddr())

	msg := []byte{}
	notFound := false

	if fn, ok := paths[*path]; ok {
		msg = []byte(fn())
	} else {
		notFound = true
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
		streamId,
		headersPayload,
	}

	data := Frame{
		0x0,
		0x1, // END_STREAM,
		streamId,
		msg,
	}

	return append(frameToBytes(headers), frameToBytes(data)...)
}

func getPath(frame Frame) *string {
	headers, err := decoder.DecodeFull(frame.payload)
	if err != nil {
		fmt.Printf("Error decoding frame %s: %s\n", fmt.Sprint(frame), err.Error())
		return nil
	}

	for _, header := range headers {
		if header.Name == ":path" {
			return &header.Value
		}
	}

	return nil
}
