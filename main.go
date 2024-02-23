package main

func main() {
	Get("/ping", func(req *Req) string {
		return "pong"
	})

	Get("/version", func(req *Req) string {
		return "0.0.42"
	})

	Post("/echo", func(req *Req) string {
		return req.body
	})

	Start()
}
