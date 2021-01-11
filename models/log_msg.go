package models

import (
	"github.com/jinzhu/gorm"
	"time"
)

type LogMsg struct {
	gorm.Model
	DirName    string    `json:"dir_name"`                             //系统类型，bis/nis/nws
	DeviceCode string    `gorm:"type:varchar(256)" json:"device_code"` //设备编码
	FaultMsg   string    `gorm:"type:text" json:"fault_msg"`           //故障信息
	DeviceImg  string    `gorm:"type:text" json:"device_img"`          //设备截图
	Status     bool      `gorm:"type:varchar(10)" json:"status"`       //故障类型，设备掉线，程序关闭，程序异常
	LogAt      string    `gorm:"type:varchar(256)" json:"log_at"`      //记录时间
	UpdateAt   time.Time `json:"update_at"`                            //记录时间
}
