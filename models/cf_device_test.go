package models

import (
	"github.com/snowlyg/LogSync/utils"
	"os"
	"testing"
)

func TestGetCfDevice(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "D:/go/src/github.com/snowlyg/LogSync")
	utils.InitConfig()
	tests := []struct {
		name    string
		want    []*CfDevice
		wantErr bool
	}{
		{
			name:    "测试获取设备数据",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCfDevice()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCfDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) == 0 {
				t.Errorf("GetCfDevice() got = %v, want %v", got, tt.want)
			}
		})
	}
}
