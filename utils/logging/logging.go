package logging

import (
	"path/filepath"
)

var RestfulLogger *Logger
var ServiceLogger *Logger
var DeviceLogger *Logger
var CommonLogger *Logger
var SyncLogger *Logger
var WorkDir string

func init() {
	RestfulLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/rest.log"))
	RestfulLogger.SetLogPrefix("log_prefix")

	ServiceLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/service.log"))
	ServiceLogger.SetLogPrefix("log_prefix")

	DeviceLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/device.log"))
	DeviceLogger.SetLogPrefix("log_prefix")

	CommonLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/common.log"))
	CommonLogger.SetLogPrefix("log_prefix")

	SyncLogger = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/sync.log"))
	SyncLogger.SetLogPrefix("log_prefix")
}
