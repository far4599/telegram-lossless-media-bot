package main

import (
	"log"
	"os"

	"github.com/far4599/telegram-lossless-media-bot/internal/app"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("DEBUG") == "true" {
		logger, err = zap.NewDevelopment()
	}
	defer logger.Sync()

	//a := app.NewApp(appID, appHash, botToken, logger)
	a := app.NewApp(logger)

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
