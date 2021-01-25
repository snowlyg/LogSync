package sync

import (
	"encoding/json"
	"fmt"
	"github.com/snowlyg/LogSync/utils"
	"os"
	"reflect"
	"testing"
	"time"
)

var location, _ = utils.GetLocation()

var timestamp = "2021-01-23 23:45:20"
var faultTxtError = &FaultTxt{"连接失败", false, timestamp}
var faultTxtSuccess = &FaultTxt{"已连接", true, timestamp}

//{"appType":"bis","call":{"code":"3","reason":"连接失败"},"face":{"code":"3","reason":"连接失败"},"interf":{"code":"3","reason":"连接失败"},"iptv":{"code":"3","reason":"连接失败"},"mqtt":{"code":"3","reason":"连接失败"},"isBackground":true,"isEmptyBed":false,"isMainActivity":false,"timestamp":"2021-01-23 23:45:20"}
var faultLogErrorBis = &FaultLog{"bis", Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"500", "连接失败"}, true, false, false, timestamp}
var faultLogErrorNis = &FaultLog{"nis", Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"500", "连接失败"}, true, false, false, timestamp}
var faultLogErrorWebApp = &FaultLog{"webapp", Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"500", "连接失败"}, true, false, false, timestamp}
var faultLogErrorNws = &FaultLog{"nws", Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"8", "连接失败"}, Plugin{"500", "连接失败"}, true, false, false, timestamp}
var faultLogSuccess = &FaultLog{"bis", Plugin{"1", "已就绪"}, Plugin{"1", "已就绪"}, Plugin{"1", "OK"}, Plugin{"1", "已就绪"}, Plugin{"200", "已就绪"}, true, false, false, timestamp}

func Test_getTimestamp(t *testing.T) {
	type args struct {
		ts string
	}

	ts := "2021-01-12 12:00:00"
	parse, _ := time.ParseInLocation(utils.DateTimeLayout, ts, location)
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name:    "空时间",
			args:    args{ts: ""},
			want:    time.Time{},
			wantErr: true,
		}, {
			name:    "正常时间",
			args:    args{ts: ts},
			want:    parse,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getTimestamp(tt.args.ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getTimestamp() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFaultTxt(t *testing.T) {
	type args struct {
		file []byte
	}
	b, _ := json.Marshal(faultTxtError)
	tests := []struct {
		name string
		args args
		want *FaultTxt
	}{
		{
			name: "fault.txt",
			args: args{file: b},
			want: faultTxtError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFaultTxt(tt.args.file)
			if err != nil {
				t.Errorf("getFaultTxt() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFaultTxt() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBoolToInt(t *testing.T) {
	type args struct {
		b bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "true",
			args: args{true},
			want: "1",
		}, {
			name: "false",
			args: args{false},
			want: "2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBoolToInt(tt.args.b); got != tt.want {
				t.Errorf("getBoolToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFaultLog(t *testing.T) {
	type args struct {
		file []byte
	}
	b, _ := json.Marshal(faultLogSuccess)
	tests := []struct {
		name string
		args args
		want *FaultLog
	}{
		{
			name: "fault.log",
			args: args{b},
			want: faultLogSuccess,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFaultLog(tt.args.file)
			if err != nil {
				t.Errorf("getFaultLog() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFaultLog() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_codeIsError(t *testing.T) {
	type args struct {
		code string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "1",
			args: args{"1"},
			want: false,
		}, {
			name: "200",
			args: args{"200"},
			want: false,
		}, {
			name: "999",
			args: args{"999"},
			want: false,
		}, {
			name: "4",
			args: args{"4"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codeIsError(tt.args.code); got != tt.want {
				t.Errorf("codeIsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getPluginsInfo_Text(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	type args struct {
		fileName string
		file     []byte
		logMsg   *LogMsg
	}
	bTextError, _ := json.Marshal(faultTxtError)
	bTexgtSuccess, _ := json.Marshal(faultTxtSuccess)
	tests := []struct {
		name string
		args args
		want struct {
			status    bool
			mqtt      string
			reason    string
			timestamp string
		}
	}{
		{
			name: "fault_txt_error",
			args: args{"fault.txt", bTextError, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      string
				reason    string
				timestamp string
			}{false, "false", "连接失败", timestamp},
		}, {
			name: "fault_txt_success",
			args: args{"fault.txt", bTexgtSuccess, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      string
				reason    string
				timestamp string
			}{true, "true", "已连接", timestamp},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getPluginsInfo(tt.args.fileName, tt.args.file, tt.args.logMsg); err != nil {
				t.Errorf("getPluginsInfo() error = %v", err)
			}
			if tt.args.logMsg.Status != tt.want.status {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Status, tt.want.status)
			}
			statusMsg := ""
			reason := tt.want.reason
			if !tt.want.status {
				reason = fmt.Sprintf("【%s】插件(mqtt): %s", utils.Config.Faultmsg.Plugin, tt.want.reason)
				statusMsg = reason
			}
			if tt.args.logMsg.Mqtt != tt.want.reason {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Mqtt, tt.want.reason)
			}
			if tt.args.logMsg.Timestamp != tt.want.timestamp {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Timestamp, tt.want.timestamp)
			}
			if tt.args.logMsg.StatusMsg != statusMsg {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.StatusMsg, statusMsg)
			}

		})
	}
}

func Test_getPluginsInfo_Log(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	type args struct {
		fileName string
		file     []byte
		logMsg   *LogMsg
	}
	type plugin struct {
		code   string
		reason string
	}
	bLogErrorNis, _ := json.Marshal(faultLogErrorNis)
	bLogErrorBis, _ := json.Marshal(faultLogErrorBis)
	bLogErrorWebApp, _ := json.Marshal(faultLogErrorWebApp)
	bLogErrorNws, _ := json.Marshal(faultLogErrorNws)
	bLogSuccess, _ := json.Marshal(faultLogSuccess)
	tests := []struct {
		name string
		args args
		want struct {
			status    bool
			mqtt      plugin
			call      plugin
			face      plugin
			interf    plugin
			iptv      plugin
			timestamp string
		}
	}{
		{
			name: "fault_log_success",
			args: args{"fault.log", bLogSuccess, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      plugin
				call      plugin
				face      plugin
				interf    plugin
				iptv      plugin
				timestamp string
			}{true, plugin{"1", "已就绪"}, plugin{"1", "已就绪"}, plugin{"1", "已就绪"}, plugin{"200", "OK"}, plugin{"1", "已就绪"}, timestamp},
		}, {
			name: "fault_log_error_nis",
			args: args{"fault.log", bLogErrorNis, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      plugin
				call      plugin
				face      plugin
				interf    plugin
				iptv      plugin
				timestamp string
			}{false, plugin{"500", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, timestamp},
		}, {
			name: "fault_log_error_bis",
			args: args{"fault.log", bLogErrorBis, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      plugin
				call      plugin
				face      plugin
				interf    plugin
				iptv      plugin
				timestamp string
			}{false, plugin{"500", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, timestamp},
		}, {
			name: "fault_log_error_webapp",
			args: args{"fault.log", bLogErrorWebApp, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      plugin
				call      plugin
				face      plugin
				interf    plugin
				iptv      plugin
				timestamp string
			}{false, plugin{"500", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, timestamp},
		}, {
			name: "fault_log_error_nws",
			args: args{"fault.log", bLogErrorNws, &LogMsg{Status: true}},
			want: struct {
				status    bool
				mqtt      plugin
				call      plugin
				face      plugin
				interf    plugin
				iptv      plugin
				timestamp string
			}{false, plugin{"500", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, plugin{"8", "连接失败"}, timestamp},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getPluginsInfo(tt.args.fileName, tt.args.file, tt.args.logMsg); err != nil {
				t.Errorf("getPluginsInfo() error = %v", err)
			}
			if tt.args.logMsg.Status != tt.want.status {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Status, tt.want.status)
			}
			statusMsg := fmt.Sprintf("【%s】", utils.Config.Faultmsg.Plugin)
			if !tt.want.status {
				if tt.args.logMsg.DevType != 3 {
					statusMsg += fmt.Sprintf("插件(mqtt): (%s)%s;", tt.want.mqtt.code, tt.want.mqtt.reason)
				}
				// 护士站主机,门旁没有iptv,interf
				if tt.args.logMsg.DevType != 3 && tt.args.logMsg.DevType != 4 {
					statusMsg += fmt.Sprintf("插件(interf): (%s)%s;", tt.want.interf.code, tt.want.interf.reason)
					statusMsg += fmt.Sprintf("插件(iptv): (%s)%s;", tt.want.iptv.code, tt.want.iptv.reason)
				}
				if tt.args.logMsg.DevType != 4 {
					statusMsg += fmt.Sprintf("插件(face): (%s)%s;", tt.want.face.code, tt.want.face.reason)
				}
				statusMsg += fmt.Sprintf("插件(call): (%s)%s;", tt.want.call.code, tt.want.call.reason)

			}
			if tt.args.logMsg.Mqtt != tt.want.mqtt.reason {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Mqtt, tt.want.mqtt)
			}
			if tt.args.logMsg.Call != tt.want.call.reason {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Mqtt, tt.want.call)
			}
			if tt.args.logMsg.Face != tt.want.face.reason {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Mqtt, tt.want.face)
			}
			if tt.args.logMsg.Interf != tt.want.interf.reason {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Mqtt, tt.want.interf)
			}
			if tt.args.logMsg.Iptv != tt.want.iptv.reason {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Mqtt, tt.want.iptv)
			}
			if tt.args.logMsg.Timestamp != tt.want.timestamp {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.Timestamp, tt.want.timestamp)
			}
			if tt.args.logMsg.StatusMsg != statusMsg {
				t.Errorf("getPluginsInfo() = %v, want %v", tt.args.logMsg.StatusMsg, statusMsg)
			}

		})
	}
}

func Test_checkSyncTime(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	type args struct {
		timetxt string
		txtTime time.Time
	}
	txtTimeFalse, _ := time.ParseInLocation(utils.DateTimeLayout, "2021-01-23 08:05:59", location)
	txtTimeTrue, _ := time.ParseInLocation(utils.DateTimeLayout, "2021-01-23 08:04:49", location)
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 int64
	}{
		{
			name:  "日志时间和服务器同步",
			args:  args{"2021-01-23 08:00:00", txtTimeTrue},
			want:  true,
			want1: 5,
		}, {
			name:  "日志时间和服务器时间不同步",
			args:  args{"2021-01-23 08:00:00", txtTimeFalse},
			want:  false,
			want1: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := checkSyncTime(tt.args.timetxt, tt.args.txtTime)
			if err != nil {
				t.Errorf("checkSyncTime() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("checkSyncTime() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkSyncTime() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkOverTime(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	type args struct {
		timeTxt string
	}
	txtTimeFalse := time.Now().In(location).Add(-15 * time.Minute).Format(utils.DateTimeLayout)
	txtTimeTrue := time.Now().In(location).Add(-14 * time.Minute).Format(utils.DateTimeLayout)
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 int64
	}{
		{
			name:  "日志时间超时",
			args:  args{txtTimeFalse},
			want:  true,
			want1: 16,
		}, {
			name:  "日志时间正常",
			args:  args{txtTimeTrue},
			want:  false,
			want1: 15,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := checkOverTime(tt.args.timeTxt)
			if err != nil {
				t.Errorf("checkOverTime() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("checkOverTime() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkOverTime() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getDeviceByCode(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	type args struct {
		remoteDevices []*utils.Device
		code          string
	}
	devices, _ := utils.GetDevices()
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "通过 code 获取远程设备信息",
			args: args{devices, "A4580F48337E"},
			want: "A4580F48337E",
		}, {
			name: "通过 code 获取远程设备信息",
			args: args{devices, "A4580F48337E"},
			want: "A4580F48337E",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDeviceByCode(tt.args.remoteDevices, tt.args.code); got != nil && got.DevCode != tt.want {
				t.Errorf("getDeviceByCode() = %v, want %v", got.DevCode, tt.want)
			}
		})
	}
}
