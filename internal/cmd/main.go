package main

import (
	"log"

	"github.com/ferux/flightcontrolcenter/internal/config"
)

func main() {
	cfg, err := config.Parse("./config.json")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%#v", cfg)
	log.Print(cfg.HTTP.Timeout.String())
}
