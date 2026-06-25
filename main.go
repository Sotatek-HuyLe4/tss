package main

import (
	"flag"
	_ "net/http/pprof"

	httpserver "github.com/bnb-chain/tss/http-server"
)

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6062", nil))
	// }()

	// cmd.Execute()

	port := flag.Int("port", 8000, "port to listen on")
	flag.Parse()

	httpserver.StartServer(*port)
}
