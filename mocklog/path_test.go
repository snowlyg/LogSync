package main

import (
	"reflect"
	"testing"
)

func TestGetPath(t *testing.T) {
	logPaths := []string{
		"/Users/snowlyg/ftp/admin/bis/A4580F450042",
		"/Users/snowlyg/ftp/admin/bis/A4580F46AB32",
		"/Users/snowlyg/ftp/admin/bis/A4580F46AB33",
		"/Users/snowlyg/ftp/admin/bis/A4580F48337F",
		"/Users/snowlyg/ftp/admin/bis/A4580F48337E",
	}
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "日志文件路径集合",
			want: logPaths,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPath(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
