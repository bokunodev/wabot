package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	_ "modernc.org/sqlite"

	"github.com/mdp/qrterminal"
	zlog "github.com/rs/zerolog/log"

	"github.com/bokunodev/wabot/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGILL)
	defer cancel()

	container, err := sqlstore.New("sqlite", "file:database.sqlite?_foreign_keys=on", log.New())
	if err != nil {
		panic(err)
	}

	store, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(store, log.New())
	defer client.Disconnect()

	defer client.RemoveEventHandler(
		client.AddEventHandler(eventhandler(ctx, client, cancel)))

	client.Connect()

	<-ctx.Done()
}

func eventhandler(ctx context.Context, client *whatsmeow.Client, disconnectCb func()) func(evt any) {
	qrch := make(chan []string)
	nextch := make(chan struct{})

	go func(nextch chan struct{}, qrch <-chan []string) {
		for codes := range qrch {
		loop2:
			for _, code := range codes {
				qrterminal.GenerateHalfBlock(code, qrterminal.L, os.Stdout)
				select {
				case <-nextch:
					break loop2
				case <-time.After(10 * time.Second):
				}
			}
		}
	}(nextch, qrch)

	return func(evt any) {
		switch ev := evt.(type) {
		case *events.QR:
			qrch <- ev.Codes
		case *events.Message:
			reply(ctx, client)(ev)
		case *events.Disconnected:
			zlog.Info().Msg("client disconnected")
			disconnectCb()
		case *events.Connected:
			zlog.Info().Msg("connected")
			nextch <- struct{}{}
		}
	}
}
