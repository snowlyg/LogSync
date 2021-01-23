package models

import (
	"github.com/snowlyg/LogSync/utils"
	"os"
	"reflect"
	"testing"
)

func TestGetCfDevice(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	tests := []struct {
		name    string
		want    []*CfDevice
		wantErr bool
	}{
		{
			name:    "测试获取设备数据",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCfDevice()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCfDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCfDevice() got = %v, want %v", got, tt.want)
			}
		})
	}
}
