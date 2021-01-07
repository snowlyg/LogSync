package logging

import (
	"github.com/snowlyg/LogSync/utils"
	"path/filepath"
)

var restfulLogger *Logger
var serviceLogger *Logger
var deviceLogger *Logger
var commonLogger *Logger
var syncLogger *Logger

//var WorkDir string

func GetRestfulLogger() *Logger {
	if restfulLogger != nil {
		return restfulLogger
	}
	workDir := getWorkDir()
	restfulLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(workDir, "./logs/rest.log"))
	restfulLogger.SetLogPrefix("log_prefix")

	return restfulLogger
}

func GetServiceLogger() *Logger {
	if serviceLogger != nil {
		return serviceLogger
	}
	workDir := getWorkDir()
	serviceLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(workDir, "./logs/service.log"))
	serviceLogger.SetLogPrefix("log_prefix")

	return serviceLogger
}

func GetDeviceLogger() *Logger {
	if deviceLogger != nil {
		return deviceLogger
	}
	workDir := getWorkDir()
	deviceLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(workDir, "./logs/device.log"))
	deviceLogger.SetLogPrefix("log_prefix")

	return deviceLogger
}

func GetCommonLogger() *Logger {
	if commonLogger != nil {
		return commonLogger
	}
	workDir := getWorkDir()
	commonLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(workDir, "./logs/common.log"))
	commonLogger.SetLogPrefix("log_prefix")

	return commonLogger
}

func GetSyncLogger() *Logger {
	if syncLogger != nil {
		return syncLogger
	}
	workDir := getWorkDir()
	syncLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(workDir, "./logs/sync.log"))
	syncLogger.SetLogPrefix("log_prefix")

	return syncLogger
}

func getWorkDir() string {
	return utils.Config.Outdir
}
