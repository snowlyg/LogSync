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
		name string
		args args
		want string
	}{
		{
			name: "nis",
			args: args{deviceTypeId: 1},
			want: "nis",
		}, {
			name: "bis",
			args: args{deviceTypeId: 2},
			want: "bis",
		}, {
			name: "webapp",
			args: args{deviceTypeId: 3},
			want: "webapp",
		}, {
			name: "nws",
			args: args{deviceTypeId: 4},
			want: "nws",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDeviceDir(tt.args.deviceTypeId)
			if err != nil {
				t.Errorf("GetDeviceDir() error = %v", err)
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

func TestGetDeviceTypeId(t *testing.T) {
	type args struct {
		deviceDir string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "nis",
			args: args{deviceDir: "nis"},
			want: 1,
		}, {
			name: "bis",
			args: args{deviceDir: "bis"},
			want: 2,
		}, {
			name: "webapp",
			args: args{deviceDir: "webapp"},
			want: 3,
		}, {
			name: "nws",
			args: args{deviceDir: "nws"},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDeviceTypeId(tt.args.deviceDir)
			if err != nil {
				t.Errorf("GetDeviceTypeId() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetDeviceTypeId() got = %v, want %v", got, tt.want)
			}
		})
	}
}
