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
	logDir := "logs"
	logFileName := "push.log"
	logFilePath := filepath.Join(currentDir, logDir, logFileName)
	// os.O_APPEND
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		panic(err)
	}
	return log.New(logFile, "pushlogger:", log.LstdFlags)
}
