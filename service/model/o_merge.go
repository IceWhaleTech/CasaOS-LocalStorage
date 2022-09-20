package model

import "gorm.io/gorm"

// Merge
type Merge struct {
	gorm.Model
	MountPoint  string        `json:"mount_point" gorm:"uniqueIndex,check:mount_point<>''"`
	BasePath    *string       `json:"base_path"`
	SerialDisks []*SerialDisk `gorm:"many2many:o_merge_disk;"`
}

func (p *Merge) TableName() string {
	return "o_merge"
}
