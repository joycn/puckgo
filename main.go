package main

import (
	"github.com/joycn/puckgo/cmd"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	cmd.Execute()
}
