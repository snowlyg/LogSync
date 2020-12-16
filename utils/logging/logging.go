package logging

var Dbug *Logger
var Err *Logger
var Norm *Logger

func init() {
	Dbug = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, "./logs/debug.log")
	Dbug.SetLogPrefix("log_prefix")

	Err = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, "./logs/error.log")
	Err.SetLogPrefix("log_prefix")

	Norm = NewLogger(&Options{
		Rolling:     DAILY,
		TimesFormat: TIMESECOND,
	}, "./logs/info.log")
	Norm.SetLogPrefix("log_prefix")
}
