## What is this?
A web server and an implementation of the HTTP/2 over TCP protocol, written in Go.

## Getting started
Define endpoints in `main.go` and start the server:
```sh
cd h2c
go run .
```

To execute a single request:
```sh
curl --http2 http://localhost:80/ping
```

To execute multiple requests in the same session:
```sh
curl --http2 --config urls.txt
```

## Frames
Example of a HTTP/2 SETTINGS frame coming from a client:\
50 52 49 20 2a 20 48 54 54 50 2f 32 2e 30 0d 0a 0d 0a 53 4d 0d 0a 0d 0a - "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n", connection preface\
00 00 12 - payload length; 18 bytes\
04 - frame type; SETTINGS\
00 - frame flags; none\
00 00 00 00 - stream identifier; for the whole connection in this case\
00 03 00 00 00 64 | 00 04 00 a0 00 00 | 00 02 00 00 00 00 - payload; 3 settings, 2 bytes for setting id, 4 for a value\

Actual settings:\
SETTINGS_MAX_CONCURRENT_STREAMS (0x3) = 0x64 = 100\
SETTINGS_INITIAL_WINDOW_SIZE (0x4) = 0x00a00000 = 10485760 bytes\
SETTINGS_ENABLE_PUSH (0x2) = 0x0 = disabled\

Wireshark (for example) supports visualization of frames from this protocol.
