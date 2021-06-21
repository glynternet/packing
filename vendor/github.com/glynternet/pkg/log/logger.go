package log

import (
	"io"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const levelKey = "level"

func NewLogger(w io.Writer) Logger {
	return logger{
		Logger: log.With(
			level.NewFilter(
				level.NewInjector(log.NewJSONLogger(w), level.InfoValue()),
				level.AllowDebug()),
			"ts", log.DefaultTimestampUTC,
			"caller", log.Caller(4)),
	}
}

type Logger interface {
	Log(...KV) error
}

type logger struct {
	log.Logger
}

func (l logger) Log(kvs ...KV) error {
	return l.Logger.Log(kvsAsParams(kvs)...)
}

func Debug(logger Logger, kvs ...KV) error {
	return logger.Log(append(
		kvs,
		KV{K: levelKey, V: "debug"},
	)...)
}

func Info(logger Logger, kvs ...KV) error {
	return logger.Log(append(
		kvs,
		KV{K: levelKey, V: "info"},
	)...)
}

func Warn(logger Logger, kvs ...KV) error {
	return logger.Log(append(
		kvs,
		KV{K: levelKey, V: "warn"},
	)...)
}

func Error(logger Logger, kvs ...KV) error {
	return logger.Log(append(
		kvs,
		KV{K: levelKey, V: "error"},
	)...)
}

type KV struct {
	K string
	V interface{}
}

type kvs []KV

func kvsAsParams(kvs kvs) []interface{} {
	ps := make([]interface{}, 2*len(kvs))
	for i, kv := range kvs {
		ps[i*2], ps[i*2+1] = kv.K, kv.V
	}
	return ps
}

func Message(msg string) KV {
	return KV{
		K: "message",
		V: msg,
	}
}

func ErrorMessage(err error) KV {
	return KV{
		K: "error",
		V: err,
	}
}
