package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mdp/qrterminal/v3"
	zlog "github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	_ "modernc.org/sqlite"

	"github.com/bokunodev/wabot/log"
)

var config string

func init() {
	flag.StringVar(&config, "config", "", "path to config.toml")
	flag.Parse()

	if config == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	f, err := os.OpenFile(config, os.O_CREATE|os.O_RDWR, os.ModePerm|0o644)
	if err != nil {
		panic(err)
	}

	stat, err := f.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Size() == 0 {
		if err = toml.NewEncoder(f).Encode(cfg); err != nil {
			panic(err)
		}

		if _, err = f.Seek(0, io.SeekStart); err != nil {
			panic(err)
		}
	}

	if _, err = toml.NewDecoder(f).Decode(&cfg); err != nil {
		panic(err)
	}

	if !cfg.Enable {
		fmt.Printf("set `Enable` to true in config `%s`\n", config)
		os.Exit(0)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGILL)
	defer cancel()

	container, err := sqlstore.New("sqlite",
		fmt.Sprintf("file:%s?_foreign_keys=on", cfg.DBFile), log.New())
	if err != nil {
		panic(err)
	}

	store, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(store, log.New())

	defer client.RemoveEventHandler(
		client.AddEventHandler(evhandler(ctx, cancel, client)))

	if err = client.Connect(); err != nil {
		panic(err)
	}
	defer client.Disconnect()

	<-ctx.Done()
}

var cfg = Config{
	TikTokAPI: "http://139.59.117.150:2020/api/tk",
	DBFile:    "database.sqlite",
	Enable:    false,
}

type Config struct {
	TikTokAPI string
	DBFile    string
	Enable    bool
}

func evhandler(ctx context.Context, cancel func(), client *whatsmeow.Client) func(any) {
	qrch := make(chan []string, 1)
	nextch := make(chan struct{}, 1)

	go func(ctx context.Context) {
		for codes := range qrch {
		secondloop:
			for _, code := range codes {
				qrterminal.GenerateHalfBlock(code, qrterminal.M, os.Stdout)
				select {
				case <-nextch:
					break secondloop
				case <-ctx.Done():
					return
				case <-time.After(20 * time.Second):
				}
			}
		}
	}(ctx)

	return func(evt any) {
		switch ev := evt.(type) {
		case *events.Message:
			if ev.Message.Conversation == nil {
				zlog.Debug().Msg("received a non text message; ignored")
				return
			}

			zlog.Info().Str("url", *ev.Message.Conversation).Send()
			values := url.Values{}
			values.Set("url", *ev.Message.Conversation)

			res, err := http.Post(cfg.TikTokAPI, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
			if err != nil {
				zlog.Error().Err(err).Send()
				return
			}
			defer res.Body.Close()

			var data TikTokAPIResponse
			if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
				zlog.Error().Err(err).Send()
				return
			}

			if data.Status != "success" {
				zlog.Error().Msg("failed")
				return
			}

			switch data.Type {
			case "video":
				dl, err := http.Get(data.URLDownload)
				if err != nil {
					zlog.Error().Err(err).Send()
					return
				}

				defer dl.Body.Close()
				dp, err := io.ReadAll(dl.Body)
				if err != nil {
					zlog.Error().Err(err).Send()
					return
				}

				err = sendVideo(ctx, client, ev.Info.Chat.User, data.Description, dp)
				if err != nil {
					zlog.Error().Err(err).Send()
					return
				}
			default:
				zlog.Error().Any("data", data).Msg("mediatype not supported")
				return
			}

		case *events.QR:
			zlog.Info().Msg("qrcode")
			select {
			case qrch <- ev.Codes:
			case <-ctx.Done():
				return
			}
		case *events.Connected:
			nextch <- struct{}{}
			zlog.Info().Msg("connected")
		case *events.PairError:
			nextch <- struct{}{}
			zlog.Info().Msg("pair error")
		case *events.LoggedOut:
			cancel()
			zlog.Warn().Msg("logged out")
		}
	}
}

type TikTokAPIResponse struct {
	Status      string `json:"status"`
	Type        string `json:"type"`
	URLDownload string `json:"urlDownload"`
	Description string `json:"description"`
}

func sendVideo(ctx context.Context, client *whatsmeow.Client, to string, caption string, b []byte) error {
	ur, err := client.Upload(ctx, b, whatsmeow.MediaVideo)
	if err != nil {
		return err
	}

	message := &proto.Message{
		VideoMessage: &proto.VideoMessage{
			Url:           &ur.URL,
			Mimetype:      toPtr("video/mp4"),
			FileSha256:    ur.FileSHA256,
			FileLength:    &ur.FileLength,
			MediaKey:      ur.MediaKey,
			Caption:       &caption,
			FileEncSha256: ur.FileEncSHA256,
			DirectPath:    &ur.DirectPath,
		},
	}

	_, err = client.SendMessage(ctx, types.NewJID(to, types.DefaultUserServer), message)
	if err != nil {
		return err
	}

	return nil
}

func toPtr[T any](v T) *T {
	return &v
}
