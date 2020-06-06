package models

import (
	"github.com/jinzhu/gorm"
)

type LogMsg struct {
	gorm.Model
	DirName      string //系统类型，bis/nis/nws
	HospitalCode string `gorm:"type:varchar(256)"` //医院编码
	DeviceCode   string `gorm:"type:varchar(256)"` //设备编码
	FaultMsg     string `gorm:"type:text"`         //故障信息
	LogAt        string `gorm:"type:varchar(256)"` //记录时间
	MD5          string `gorm:"type:varchar(256)"` //记录时间
}
