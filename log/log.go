package log

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	walog "go.mau.fi/whatsmeow/util/log"
)

func init() {
	_, nocolor := os.LookupEnv("LOG_NO_COLOR")

	loglvl := zerolog.DebugLevel

	if lvl, ok := os.LookupEnv("LOG_LEVEL"); ok {
		loglvl = strToLevel(lvl)
	}

	var w io.Writer = os.Stderr
	if s, ok := os.LookupEnv("LOG_FILE"); ok {
		f, err := os.OpenFile(s, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm|0o644)
		if err != nil {
			panic(err)
		}

		w = f
	}

	zlog.Logger = zlog.Logger.
		With().
		Caller().
		Logger().
		Output(zerolog.ConsoleWriter{
			Out:     w,
			NoColor: nocolor,
		}).
		Level(loglvl)
}

func strToLevel(s string) zerolog.Level {
	switch s {
	case zerolog.LevelTraceValue:
		return zerolog.TraceLevel
	case zerolog.LevelDebugValue:
		return zerolog.DebugLevel
	case zerolog.LevelInfoValue:
		return zerolog.InfoLevel
	case zerolog.LevelWarnValue:
		return zerolog.WarnLevel
	case zerolog.LevelErrorValue:
		return zerolog.ErrorLevel
	case zerolog.LevelFatalValue:
		return zerolog.FatalLevel
	case zerolog.LevelPanicValue:
		return zerolog.PanicLevel
	}

	panic("invalid value for env variable 'LOG_LEVEL'")
}

type Logger struct {
	log zerolog.Logger
}

func New() *Logger {
	return &Logger{log: zlog.Logger}
}

func (logger *Logger) Warnf(msg string, args ...interface{}) {
	zlog.Warn().Msgf(msg, args...)
}

func (logger *Logger) Errorf(msg string, args ...interface{}) {
	zlog.Error().Msgf(msg, args...)
}

func (logger *Logger) Infof(msg string, args ...interface{}) {
	zlog.Info().Msgf(msg, args...)
}

func (logger *Logger) Debugf(msg string, args ...interface{}) {
	zlog.Debug().Msgf(msg, args...)
}

func (logger *Logger) Sub(module string) walog.Logger {
	ret := New()
	ret.log = ret.log.With().Str("module", module).Logger()

	return ret
}
