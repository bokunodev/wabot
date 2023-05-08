package main

import (
	"context"
	"io"
	"net/http"
	"os"

	zlog "github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func reply(ctx context.Context, client *whatsmeow.Client) func(*events.Message) {
	return func(ev *events.Message) {
		if ev == nil {
			panic(ev)
		}

		zlog.Debug().Msg("message received")

		// p, err := getcat()
		p, err := os.ReadFile("/tmp/vid.mp4")
		if err != nil {
			zlog.Error().Err(err).Send()
			return
		}

		res, err := client.Upload(ctx, p, whatsmeow.MediaVideo)
		if err != nil {
			zlog.Error().Err(err).Send()
			return
		}

		message := &proto.Message{
			// Conversation: ptrTo("gambar koceng"),
			VideoMessage: &proto.VideoMessage{
				Url:           &res.URL,
				Mimetype:      ptrTo("video/mp4"),
				Caption:       ptrTo("joget slur"),
				FileSha256:    res.FileSHA256,
				FileEncSha256: res.FileEncSHA256,
				FileLength:    &res.FileLength,
				MediaKey:      res.MediaKey,
				DirectPath:    &res.DirectPath,
			},
		}

		_, err = client.SendMessage(ctx, types.NewJID(ev.Info.Chat.User, types.DefaultUserServer), message)
		if err != nil {
			zlog.Error().Err(err).Send()
			return
		}

		zlog.Debug().Msg("message sent")
	}
}

func ptrTo[T any](v T) *T {
	return &v
}

func getcat() ([]byte, error) {
	res, err := http.Get("https://cataas.com/cat")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}
