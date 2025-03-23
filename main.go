package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalln("error loading .env file")
	}

	lastFmApiKey := os.Getenv("LAST_FM_API_KEY")
	if lastFmApiKey == "" {
		log.Fatalln("LAST_FM_API_KEY not set")
	}

	lastFmUser := os.Getenv("LAST_FM_USER")
	if lastFmApiKey == "" {
		log.Fatalln("LAST_FM_USER not set")
	}

	lastFm, err := InitLastFm(lastFmUser, lastFmApiKey)
	if err != nil {
		log.Fatalln("error initializing LastFm struct: " + err.Error())
	}

	lovedTracks, err := lastFm.GetLovedTracks()
	if err != nil {
		log.Fatalln("error getting loved tracks: " + err.Error())
	}

	for i, t := range lovedTracks {
		fmt.Println(i+1, t.Artist.Name, t.Name)
	}
}
