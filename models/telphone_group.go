package models

type TelphoneGroup struct {
	TelgId       int64  `json:"telg_id" gorm:"column:telg_id"`                       // id
	TelgCode     int64  `json:"telg_code" gorm:"column:telg_code"`                   // 代码
	TelgDesc     string `json:"telg_desc" gorm:"column:telg_desc;type:varchar(30)"`  // 描述
	TelgIcon     string `json:"telg_icon" gorm:"column:telg_icon;type:varchar(255)"` // 分类图标
	TelgParentId int64  `json:"telg_parent_id" gorm:"column:telg_parent_id"`         // 父级id
}
