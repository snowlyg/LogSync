package models

import (
	"github.com/jinzhu/gorm"
	"time"
)

type LogMsg struct {
	gorm.Model
	DirName      string    //系统类型，bis/nis/nws
	HospitalCode string    `gorm:"type:varchar(256)"` //医院编码
	DeviceCode   string    `gorm:"type:varchar(256)"` //设备编码
	FaultMsg     string    `gorm:"type:text"`         //故障信息
	Status       string    `gorm:"type:varchar(10)"`  //故障类型，设备掉线，程序关闭，程序异常
	LogAt        string    `gorm:"type:varchar(256)"` //记录时间
	UpdateAt     time.Time //记录时间
}
