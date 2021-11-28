package main

import (
	"context"
	"flag"
	"log"
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("provide the websocket URL")
	}
	websocketURL := flag.Args()[0]

	ctx := context.Background()
	if err := start(ctx, websocketURL, log.Default()); err != nil {
		log.Fatal(err)
	}
}
