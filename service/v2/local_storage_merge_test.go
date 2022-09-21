package v2

import (
	"testing"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

var (
	_db      *gorm.DB
	_service *LocalStorageService
)

func init() {
	_db = sqlite.GetDBByFile("file::memory:?cache=shared")

	sqlite.Hooks[sqlite.HookAfterDelete] = append(sqlite.Hooks[sqlite.HookAfterDelete], hookAfterDeleteMountPoint)

	_service = NewLocalStorageService(_db, nil)
}

func TestHookAfterDeleteSerialDisk(t *testing.T) {
	// create two serial disks in db
	expectedDisk1 := model2.Mount{
		UUID:       "85022acb-b5a2-424e-bfa9-6acb67d17cb8",
		Path:       "/dev/sda",
		State:      0,
		MountPoint: "/mnt/sda",
	}

	expectedDisk2 := model2.Mount{
		UUID:       "36c94c85-debf-49b6-9f19-866c14b3a0c6",
		Path:       "/dev/sdb",
		State:      0,
		MountPoint: "/mnt/sdb",
	}

	_db.Create(&expectedDisk1)
	_db.Create(&expectedDisk2)

	// create a merge in db, associated with two serial disks

	expectedMerge := model2.Merge{
		MountPoint: "/mnt/merge",
		SourceMounts: []*model2.Mount{
			&expectedDisk1,
			&expectedDisk2,
		},
	}

	_db.Create(&expectedMerge)

	// verify the merge is associated with two serial disks
	var actualMerges []model2.Merge
	if err := _db.Preload("SourceMounts").Find(&actualMerges).Error; err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(actualMerges), 1)

	actualMerge := actualMerges[0]
	assert.Equal(t, len(actualMerge.SourceMounts), 2)

	assert.DeepEqual(t, actualMerge, expectedMerge)

	// delete one serial disk
	if err := _db.InstanceSet("gdb", _db).Delete(&expectedDisk1).Error; err != nil {
		t.Error(err)
	}

	// check if the merge is updated
	if err := _db.Preload("SourceMounts").Find(&actualMerges).Error; err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(actualMerges), 1)

	actualMerge = actualMerges[0]
	assert.Equal(t, len(actualMerge.SourceMounts), 1)

	assert.DeepEqual(t, *actualMerge.SourceMounts[0], expectedDisk2)

	// delete the other serial disk
	if err := _db.Delete(&expectedDisk2).Error; err != nil {
		t.Error(err)
	}

	// check if the merge is updated
	if err := _db.Preload("SourceMounts").Find(&actualMerges).Error; err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(actualMerges), 1)

	actualMerge = actualMerges[0]
	assert.Equal(t, len(actualMerge.SourceMounts), 0)
}
