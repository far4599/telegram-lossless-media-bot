package app

import (
	"context"

	"github.com/far4599/telegram-lossless-media-bot/internal/service"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
}

func NewApp(logger *zap.Logger) *App {
	return &App{
		logger: logger,
	}
}

func (a *App) Run() error {
	if err := a.run(context.Background(), a.logger); err != nil {
		return err
	}

	return nil
}

func (a *App) run(ctx context.Context, log *zap.Logger) error {
	dispatcher := tg.NewUpdateDispatcher()
	opts := telegram.Options{
		Logger:        log,
		UpdateHandler: dispatcher,
	}
	return telegram.BotFromEnvironment(ctx, opts, func(ctx context.Context, client *telegram.Client) error {
		api := tg.NewClient(client)

		h := service.NewMessageHandler(api, a.logger)

		dispatcher.OnNewMessage(h.OnNewMessage)
		dispatcher.OnNewChannelMessage(h.OnNewChannelMessage)

		return nil
	}, telegram.RunUntilCanceled)
}
