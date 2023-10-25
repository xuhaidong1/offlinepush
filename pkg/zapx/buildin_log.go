package zapx

import (
	"log"
	"os"
	"path/filepath"
)

func NewPushLogger() *log.Logger {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	logDir := filepath.Join(currentDir, "logs")
	err = os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	logFilePath := filepath.Join(logDir, "push.log")
	// os.O_APPEND
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		panic(err)
	}
	return log.New(logFile, "pushlogger:", log.LstdFlags)
}
