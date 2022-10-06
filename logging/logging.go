package logging

import "go.uber.org/zap"

var (
	log *zap.Logger
	err error
)

func init() {
	log, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

func Logger() *zap.Logger {
	return log
}
