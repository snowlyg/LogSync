package utils

import (
	"fmt"
	"github.com/jinzhu/configor"
)

var Config = struct {
	Host      string `default:"" env:"host"`
	Appid     string `default:"" env:"appid"`
	Appsecret string `default:"" env:"appsecret"`
	DB        string `default:"" env:"db"`
	Dir       struct {
		Names string `default:"" env:"DirNames"`
		Codes string `default:"" env:"DirCodes"`
	}
	Exts        string `default:"" env:"exts"`
	Imgexts     string `default:"" env:"imagexts"`
	Root        string `default:"" env:"root"`
	Outdir      string `default:"" env:"outdir"`
	Timeout     int64  `default:"" env:"timeout"`
	Timeover    int64  `default:"" env:"timeover"`
	Isresizeimg bool   `default:"false" env:"isresizeimg"`
	Devicesize  int    `default:"5" env:"devicesize"`
	Ftp         struct {
		Ip       string `default:"" env:"FtpIp"`
		Username string `default:"" env:"FtpUsername"`
		Password string `default:"" env:"FtpPassword"`
	}
	Device struct {
		Timetype     string `default:"m" env:"DeviceTimetype"`
		Timeduration int64  `default:"3" env:"DeviceTimeduration"`
	}
	Data struct {
		Timetype     string `default:"h" env:"DataTimetype"`
		Timeduration int64  `default:"1" env:"DataTimeduration"`
	}
	Restful struct {
		Timetype     string `default:"m" env:"RestfulTimetype"`
		Timeduration int64  `default:"3" env:"RestfulTimeduration"`
	}
	Web struct {
		Account  string `default:"" env:"WebAccount"`
		Password string `default:"" env:"WebPassword"`
		Indir    string `default:"" env:"WebIndir"`
	}
	Android struct {
		Account  string `default:"" env:"AndroidAccount"`
		Password string `default:"" env:"AndroidPassword"`
		Indir    string `default:"" env:"AndroidIndir"`
	}
	Faultmsg struct {
		Device  string `default:"" env:"FaultmsgDevice"`
		Plugin  string `default:"" env:"FaultmsgPlugin"`
		Logsync string `default:"" env:"FaultmsgLogsync"`
	}
	Log struct {
		Overtime int64 `default:"15" env:"LogOvertime"`
		Synctime int64 `default:"5" env:"LogSynctime"`
	}
}{}

func InitConfig() {
	if err := configor.Load(&Config, ConfigFile()); err != nil {
		panic(fmt.Sprintf("Config Path:%s ,Error:%+v\n", ConfigFile(), err))
	}
}
