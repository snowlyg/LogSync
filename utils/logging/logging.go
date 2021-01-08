package logging

import (
	"fmt"
	"github.com/snowlyg/LogSync/utils"
	"path/filepath"
)

func GetMyLogger(name string) *Logger {
	var logger *Logger
	workDir := getWorkDir()
	logger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(workDir, fmt.Sprintf("./logs/%s.log", name)))
	logger.SetLogPrefix("log_prefix")
	return logger
}

func getWorkDir() string {
	return utils.Config.Outdir
}
