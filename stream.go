package main

import "net"

var streams = make(map[string]map[int]*Req)

func addStreamReq(conn net.Conn, streamId int, req *Req) {
	addrStr := conn.RemoteAddr().String()
	if _, ok := streams[addrStr]; !ok {
		streams[addrStr] = make(map[int]*Req)
	}

	streams[addrStr][streamId] = req
}

func getStreamReq(conn net.Conn, streamId int) *Req {
	addrStr := conn.RemoteAddr().String()
	if _, ok := streams[addrStr]; !ok {
		return nil
	}

	return streams[addrStr][streamId]
}

func delStreamReq(conn net.Conn, streamId int) {
	addrStr := conn.RemoteAddr().String()
	if _, ok := streams[addrStr]; !ok {
		return
	}

	delete(streams[addrStr], streamId)
}
