package logging

import (
	"path/filepath"
)

var Dbug *Logger
var Err *Logger
var Norm *Logger
var WorkDir string

func init() {
	Dbug = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/debug.log"))
	Dbug.SetLogPrefix("log_prefix")

	Err = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/error.log"))
	Err.SetLogPrefix("log_prefix")

	Norm = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, filepath.Join(WorkDir, "./logs/info.log"))
	Norm.SetLogPrefix("log_prefix")
}
