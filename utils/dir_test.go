package utils

import (
	"os"
	"reflect"
	"testing"
)

func TestGetDeviceDir(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	InitConfig()
	//names: nis,bis,webapp,nws
	//codes: 1,2,3,4
	type args struct {
		deviceTypeId int64
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "nis",
			args:    args{deviceTypeId: 1},
			want:    "nis",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDeviceDir(tt.args.deviceTypeId)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDeviceDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDirs(t *testing.T) {
	os.Setenv("LogSyncConfigPath", "/Users/snowlyg/go/src/github.com/snowlyg/LogSync")
	InitConfig()
	want := map[string]string{
		"1": "nis",
		"2": "bis",
		"3": "webapp",
		"4": "nws",
	}
	tests := []struct {
		name    string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "获取设备类型编码和名称",
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDirs()
			if (err != nil) != tt.wantErr {
				t.Errorf("getDirs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDirs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
