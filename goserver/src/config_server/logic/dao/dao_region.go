package dao

import (
	"common/proto/config"
	"gorm.io/gorm"
)

type ConfRegions struct {
	Id       int64
	Name     string
	Title    string
	ParentId int64
	Lat      float32
	Lng      float32
	OrderNum int64
	IsShow   int64
	GeoPos   string
}

func (region *ConfRegions) GetRegion(db *gorm.DB, id int64) *config.ConfRegions {
	confRegion := &ConfRegions{}
	db.Model(ConfRegions{}).Where("id = ?", id).First(&confRegion)
	regions := config.ConfRegions{
		Id:       confRegion.Id,
		Name:     confRegion.Name,
		Title:    confRegion.Title,
		ParentId: confRegion.ParentId,
		Lat:      confRegion.Lat,
		Lng:      confRegion.Lng,
		OrderNum: confRegion.OrderNum,
		IsShow:   confRegion.IsShow,
		GeoPos:   confRegion.GeoPos,
	}
	return &regions
}
