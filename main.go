package main

func main() {
	Get("/ping", func() string {
		return "pong"
	})

	Get("/version", func() string {
		return "0.0.42"
	})

	Start()
}
