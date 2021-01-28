package main

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/snowlyg/LogSync/utils"
)

func TestGetPath(t *testing.T) {
	location, _ := utils.GetLocation()
	device1 := device{"A4580F48337E", "bis", "fault.log", "interface.log"}
	device2 := device{"A4580F48337F", "bis", "fault.log", "interface.log"}
	tests := []struct {
		name string
		arg  device
		want string
	}{
		{
			name: "日志文件路径A4580F48337E",
			arg:  device1,
			want: filepath.Join("D:/env/FileZillaServer/share/log/bis/A4580F48337E", time.Now().In(location).Format(utils.DateLayout)),
		}, {
			name: "日志文件路径A4580F48337F",
			arg:  device2,
			want: filepath.Join("D:/env/FileZillaServer/share/log/bis/A4580F48337F", time.Now().In(location).Format(utils.DateLayout)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPath(tt.arg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
