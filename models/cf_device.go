package models

import "github.com/snowlyg/LogSync/utils"

//dev_id as local_device_id,
//dev_code as device_code,
//dev_desc as device_desc,
//dev_position as device_position ,
//dev_type as device_type_id,
//dev_ip as device_ip,
//dev_active as device_active,
//dev_create_time as create_at

type CfDevice struct {
	DevId         int64         `json:"local_device_id" gorm:"column:dev_id"`                        // 服务id
	DevType       utils.DirName `json:"device_type_id" gorm:"column:dev_type"`                       // 类型
	DevStatus     int64         `json:"device_status" gorm:"column:dev_status"`                      // 状态
	DevActive     int64         `json:"device_active" gorm:"column:dev_active"`                      // 状态
	DevCode       string        `json:"device_code" gorm:"column:dev_code;type:varchar(50)"`         // 设备代码
	DevDesc       string        `json:"device_desc" gorm:"column:dev_desc;type:varchar(50)"`         // 设备描述
	DevPosition   string        `json:"device_position" gorm:"column:dev_position;type:varchar(50)"` // 位置
	DevIp         string        `json:"device_ip" gorm:"column:dev_ip;type:varchar(40)"`             // ip
	LocDesc       string        `json:"loc_desc" gorm:"column:loc_desc;type:varchar(60)"`            // loc_desc
	RoomDesc      string        `json:"room_desc" gorm:"column:room_desc;type:varchar(60)"`          // room_desc
	BedCode       string        `json:"bed_code" gorm:"column:bed_code;type:varchar(255)"`           // bed_code
	DevCreateTime string        `json:"create_at" gorm:"column:dev_create_time"`                     // 创建时间
}

func GetCfDevice() ([]*CfDevice, error) {
	var cfDevices []*CfDevice
	err := utils.GetSQLite().Where("dev_status = ?", 1).Where("dev_active = ?", 1).Find(&cfDevices).Error
	defer utils.GetSQLite().Close()
	if err != nil {
		return nil, err
	}

	return cfDevices, nil
}
