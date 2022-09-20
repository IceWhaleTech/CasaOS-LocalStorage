package model

import "gorm.io/gorm"

// Merge
type Merge struct {
	gorm.Model
	Path        string `json:"path"`
	SerialDisks []SerialDisk `gorm:"many2many:o_merge_disk;"`
}

func (p *Merge) TableName() string {
	return "o_merge"
}
