package models

type ConfRegions struct {
	Id       uint   `gorm:"primaryKey;NOT NULL;AUTO_INCREMENT;COMMENT:id"`
	Name     string `gorm:"size:255;NOT NULL;default:'';COMMENT:地区name"`
	Title    string `gorm:"size:255;NOT NULL;default:'';COMMENT:地区Title"`
	ParentId int    `gorm:"size:30;NOT NULL;default:0;COMMENT:父级id"`
	OrderNum int    `gorm:"size:4;NOT NULL;default:0;COMMENT:排序"`
	IsShow   int    `gorm:"size:4;NOT NULL;default:0;COMMENT:1展示，0隐藏"`
	Lat      string `gorm:"size:255;NOT NULL;default:'';COMMENT:纬度"`
	Lng      string `gorm:"size:255;NOT NULL;default:'';COMMENT:经度"`
	GeoPos   string `gorm:"size:255;NOT NULL;default:'';COMMENT:latitude,longitude"`
}
