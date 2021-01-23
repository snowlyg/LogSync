package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CWD() string {
	if os.Getenv("LogSyncConfigPath") != "" {
		return os.Getenv("LogSyncConfigPath")
	}
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}

func ConfigFile() string {
	return filepath.Join(CWD(), "config.yaml")
}

func Exist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func IsPortInUse(host string, port int64) error {
	if conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), 3*time.Second); err == nil {
		conn.Close()
		return nil
	} else {
		return err
	}

}

func ListDir(dirPth string, suffix string) ([]string, error) {
	var files []string
	files = make([]string, 0, 10)
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	PthSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) { //匹配文件
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}
	return files, nil
}

func CreateDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}
