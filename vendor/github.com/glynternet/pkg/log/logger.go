package log

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func NewLogger() Logger {
	return logger{
		Logger: log.With(
			level.NewFilter(log.NewJSONLogger(os.Stderr), level.AllowDebug()),
			"ts", log.DefaultTimestampUTC,
			"caller", log.DefaultCaller),
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

func Error(err error) KV {
	return KV{
		K: "error",
		V: err,
	}
}
