package model

import "time"

// Merge
type Merge struct {
	ID           uint `gorm:"primarykey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MountPoint   string   `json:"mount_point" gorm:"uniqueIndex,check:mount_point<>''"`
	SourceMounts []*Mount `json:"source_mounts" gorm:"many2many:o_merge_disk;"`
}

func (p *Merge) TableName() string {
	return "o_merge"
}
