package largecache

import (
	"log"
	"os"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

var _ Logger = &log.Logger{}

func DefaultLogger() *log.Logger {
	return log.New(os.Stdout, "", log.LstdFlags)
}

func newLogger(custome Logger) Logger {
	if custome != nil {
		return custome
	}
	return DefaultLogger()
}
