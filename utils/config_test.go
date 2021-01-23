package utils

import (
	"os"
	"strconv"
	"testing"
)

func TestInitConfig(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	InitConfig()

	tests := []struct {
		name string
		kvs  []struct {
			key  string
			want string
		}
	}{
		{
			name: "配置文件测试",
			kvs: []struct {
				key  string
				want string
			}{
				{
					key:  Config.Appid,
					want: "dsfdsfdsf",
				}, {
					key:  Config.Appsecret,
					want: "sdfsdfsdfdsfsdfsdf",
				}, {
					key:  Config.DB,
					want: "visible:Chindeo@tcp(127.0.0.1:3306)/dois?charset=utf8mb4&parseTime=True&loc=Local",
				}, {
					key:  strconv.FormatInt(Config.Timeout, 10),
					want: "5",
				}, {
					key:  strconv.FormatInt(Config.Timeover, 10),
					want: "10",
				}, {
					key:  Config.Host,
					want: "test.op.com",
				}, {
					key:  Config.Exts,
					want: "fault.log,interface.log,fault.txt,error.txt",
				}, {
					key:  Config.Imgexts,
					want: "ScreenCapture.png,screen.png",
				}, {
					key:  Config.Root,
					want: "log",
				}, {
					key:  Config.Outdir,
					want: "/Users/snowlyg/go/src/github.com/snowlyg/LogSync",
				}, {
					key:  strconv.FormatBool(Config.Isresizeimg),
					want: "false",
				}, {
					key:  strconv.FormatInt(int64(Config.Devicesize), 10),
					want: "10",
				}, {
					key:  Config.Ftp.Ip,
					want: "127.0.0.1",
				}, {
					key:  Config.Ftp.Username,
					want: "admin",
				}, {
					key:  Config.Ftp.Password,
					want: "Chindeo",
				}, {
					key:  Config.Restful.Timetype,
					want: "m",
				}, {
					key:  strconv.FormatInt(Config.Restful.Timeduration, 10),
					want: "1",
				}, {
					key:  Config.Device.Timetype,
					want: "m",
				}, {
					key:  strconv.FormatInt(Config.Device.Timeduration, 10),
					want: "1",
				}, {
					key:  Config.Data.Timetype,
					want: "m",
				}, {
					key:  strconv.FormatInt(Config.Data.Timeduration, 10),
					want: "5",
				}, {
					key:  Config.Dir.Names,
					want: "nis,bis,webapp,nws",
				}, {
					key:  Config.Dir.Codes,
					want: "1,2,3,4",
				}, {
					key:  Config.Web.Account,
					want: "administrator",
				}, {
					key:  Config.Web.Password,
					want: "123456",
				}, {
					key:  Config.Web.Indir,
					want: "D:/App/data/log",
				}, {
					key:  Config.Android.Account,
					want: "root",
				}, {
					key:  Config.Android.Password,
					want: "Chindeo",
				}, {
					key:  Config.Android.Indir,
					want: "/sdcard/chindeo_app/log",
				}, {
					key:  Config.Faultmsg.Device,
					want: "设备异常",
				}, {
					key:  Config.Faultmsg.Plugin,
					want: "插件异常",
				}, {
					key:  Config.Faultmsg.Logsync,
					want: "日志同步异常",
				}, {
					key:  strconv.FormatInt(int64(Config.Log.Overtime), 10),
					want: "15",
				}, {
					key:  strconv.FormatInt(int64(Config.Log.Synctime), 10),
					want: "5",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, kv := range tt.kvs {
				if kv.key != kv.want {
					t.Errorf("Config kv want '%v' and get %v", kv.want, kv.key)
				}
			}
		})
	}
}
