package main

import (
	_ "net/http/pprof"

	httpserver "github.com/bnb-chain/tss/http-server"
)

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6062", nil))
	// }()

	// cmd.Execute()

	httpserver.StartServer()
}
