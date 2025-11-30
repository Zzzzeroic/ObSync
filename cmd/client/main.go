package main

import (
	"flag"
	"log"
	"os"

	"obsync/pkg/sync"
)

func main() {
	var (
		hub  = flag.String("hub", "localhost:9527", "Hub address")
		repo = flag.String("repo", "D:/Obsidian/testRepo", "Local repo path")
		id   = flag.String("id", "client_114514", "Client ID")
	)
	flag.Parse()

	if *id == "" {
		hostname, _ := os.Hostname()
		*id = hostname
	}

	client := sync.NewClient(*hub, *repo, *id)
	log.Fatal(client.Run())
}
