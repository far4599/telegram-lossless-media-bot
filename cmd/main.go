package main

import (
	"log"
	"os"

	"github.com/far4599/telegram-lossless-media-bot/internal/app"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("DEBUG") == "true" {
		logger, err = zap.NewDevelopment()
		if err != nil {
			log.Fatal(err)
		}
	}
	defer logger.Sync()

	if err := app.NewApp(logger).Run(); err != nil {
		log.Fatal(err)
	}
}
