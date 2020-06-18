package models

type Telphone struct {
	TelId             int64  `json:"tel_id" gorm:"column:tel_id"`                                              // id
	SsTelGroupId      int64  `json:"ss_tel_group_id" gorm:"column:ss_tel_group_id"`                            // 项目分类,急诊、维修、检查等等
	TelDesc           string `json:"tel_desc" gorm:"column:tel_desc;type:varchar(50)"`                         //通讯录名称
	TelHousePhone     string `json:"tel_house_phone" gorm:"column:tel_house_phone;type:varchar(15)"`           // 内线电话
	TelOutsidePhone   string `json:"tel_outside_phone" gorm:"column:tel_outside_phone;type:varchar(15)"`       // 外线电话
	TelLocContactName string `json:"tel_loc_contact_name" gorm:"column:tel_loc_contact_name;type:varchar(30)"` // 拼音码
}
