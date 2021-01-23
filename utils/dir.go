package utils

import (
	"errors"
	"strconv"
	"strings"
)

// bis 床旁
// nis 护理白板
// nws 护士站主机
// webapp 门旁
func getDirs() (map[string]string, error) {
	if len(Config.Dir.Names) == 0 {
		return nil, errors.New("设备类型名称配置为空")
	}
	if len(Config.Dir.Codes) == 0 {
		return nil, errors.New("设备类型代码配置为空")
	}

	names := strings.Split(Config.Dir.Names, ",")
	codes := strings.Split(Config.Dir.Codes, ",")
	if len(names) == 0 {
		return nil, errors.New("设备类型名称配置为空")
	}
	if len(codes) == 0 {
		return nil, errors.New("设备类型代码配置为空")
	}

	if len(codes) != len(names) {
		return nil, errors.New("设备类型代码和设备类型名称配置不匹配")
	}
	dirs := map[string]string{}
	for i := 0; i < len(names); i++ {
		dirs[codes[i]] = names[i]
	}
	return dirs, nil
}

// 获取日志类型目录
func GetDeviceDir(deviceTypeId int64) (string, error) {
	dirs, err := getDirs()
	if err != nil {
		return "", nil
	}
	if len(dirs) > 0 {
		for d, dir := range dirs {
			var typeid int
			typeid, err = strconv.Atoi(d)
			if err != nil {
				return "", nil
			}
			if int64(typeid) == deviceTypeId {
				return dir, nil
			}
		}
	}
	return "other", nil
}
