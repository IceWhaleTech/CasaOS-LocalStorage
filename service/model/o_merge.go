package model

import "time"

const MergeSourceVolumes = "SourceVolumes"

// Merge
type Merge struct {
	ID             uint `gorm:"primarykey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FSType         string    `json:"fstype"`
	MountPoint     string    `json:"mount_point" gorm:"uniqueIndex,check:mount_point<>''"`
	SourceBasePath *string   `json:"source_base_path"`
	SourceVolumes  []*Volume `json:"source_volumes" gorm:"many2many:o_merge_disk;"`
}

func (p *Merge) TableName() string {
	return "o_merge"
}
