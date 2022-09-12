package v2

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/wrapper"
	"gotest.tools/v3/assert"

	"github.com/moby/sys/mountinfo"
)

var _allMountInfo []mountinfo.Info

type MountInfoMock struct{}

func init() {
	_allMountInfoJSON := `
	[
		{
			"ID": 24,
			"Parent": 29,
			"Major": 0,
			"Minor": 22,
			"Root": "/",
			"Mountpoint": "/sys",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:7",
			"FSType": "sysfs",
			"Source": "sysfs",
			"VFSOptions": "rw"
		},
		{
			"ID": 25,
			"Parent": 29,
			"Major": 0,
			"Minor": 23,
			"Root": "/",
			"Mountpoint": "/proc",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:13",
			"FSType": "proc",
			"Source": "proc",
			"VFSOptions": "rw"
		},
		{
			"ID": 26,
			"Parent": 29,
			"Major": 0,
			"Minor": 5,
			"Root": "/",
			"Mountpoint": "/dev",
			"Options": "rw,nosuid,relatime",
			"Optional": "shared:2",
			"FSType": "devtmpfs",
			"Source": "udev",
			"VFSOptions": "rw,size=1972852k,nr_inodes=493213,mode=755,inode64"
		},
		{
			"ID": 27,
			"Parent": 26,
			"Major": 0,
			"Minor": 24,
			"Root": "/",
			"Mountpoint": "/dev/pts",
			"Options": "rw,nosuid,noexec,relatime",
			"Optional": "shared:3",
			"FSType": "devpts",
			"Source": "devpts",
			"VFSOptions": "rw,gid=5,mode=620,ptmxmode=000"
		},
		{
			"ID": 28,
			"Parent": 29,
			"Major": 0,
			"Minor": 25,
			"Root": "/",
			"Mountpoint": "/run",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:5",
			"FSType": "tmpfs",
			"Source": "tmpfs",
			"VFSOptions": "rw,size=401704k,mode=755,inode64"
		},
		{
			"ID": 29,
			"Parent": 1,
			"Major": 8,
			"Minor": 1,
			"Root": "/",
			"Mountpoint": "/",
			"Options": "rw,relatime",
			"Optional": "shared:1",
			"FSType": "ext4",
			"Source": "/dev/sda1",
			"VFSOptions": "rw"
		},
		{
			"ID": 30,
			"Parent": 24,
			"Major": 0,
			"Minor": 6,
			"Root": "/",
			"Mountpoint": "/sys/kernel/security",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:8",
			"FSType": "securityfs",
			"Source": "securityfs",
			"VFSOptions": "rw"
		},
		{
			"ID": 31,
			"Parent": 26,
			"Major": 0,
			"Minor": 26,
			"Root": "/",
			"Mountpoint": "/dev/shm",
			"Options": "rw,nosuid,nodev",
			"Optional": "shared:4",
			"FSType": "tmpfs",
			"Source": "tmpfs",
			"VFSOptions": "rw,inode64"
		},
		{
			"ID": 32,
			"Parent": 28,
			"Major": 0,
			"Minor": 27,
			"Root": "/",
			"Mountpoint": "/run/lock",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:6",
			"FSType": "tmpfs",
			"Source": "tmpfs",
			"VFSOptions": "rw,size=5120k,inode64"
		},
		{
			"ID": 33,
			"Parent": 24,
			"Major": 0,
			"Minor": 28,
			"Root": "/",
			"Mountpoint": "/sys/fs/cgroup",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:9",
			"FSType": "cgroup2",
			"Source": "cgroup2",
			"VFSOptions": "rw,nsdelegate,memory_recursiveprot"
		},
		{
			"ID": 34,
			"Parent": 24,
			"Major": 0,
			"Minor": 29,
			"Root": "/",
			"Mountpoint": "/sys/fs/pstore",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:10",
			"FSType": "pstore",
			"Source": "pstore",
			"VFSOptions": "rw"
		},
		{
			"ID": 35,
			"Parent": 24,
			"Major": 0,
			"Minor": 30,
			"Root": "/",
			"Mountpoint": "/sys/firmware/efi/efivars",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:11",
			"FSType": "efivarfs",
			"Source": "efivarfs",
			"VFSOptions": "rw"
		},
		{
			"ID": 36,
			"Parent": 24,
			"Major": 0,
			"Minor": 31,
			"Root": "/",
			"Mountpoint": "/sys/fs/bpf",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:12",
			"FSType": "bpf",
			"Source": "bpf",
			"VFSOptions": "rw,mode=700"
		},
		{
			"ID": 37,
			"Parent": 25,
			"Major": 0,
			"Minor": 32,
			"Root": "/",
			"Mountpoint": "/proc/sys/fs/binfmt_misc",
			"Options": "rw,relatime",
			"Optional": "shared:14",
			"FSType": "autofs",
			"Source": "systemd-1",
			"VFSOptions": "rw,fd=29,pgrp=1,timeout=0,minproto=5,maxproto=5,direct,pipe_ino=13214"
		},
		{
			"ID": 38,
			"Parent": 24,
			"Major": 0,
			"Minor": 7,
			"Root": "/",
			"Mountpoint": "/sys/kernel/debug",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:15",
			"FSType": "debugfs",
			"Source": "debugfs",
			"VFSOptions": "rw"
		},
		{
			"ID": 39,
			"Parent": 26,
			"Major": 0,
			"Minor": 33,
			"Root": "/",
			"Mountpoint": "/dev/hugepages",
			"Options": "rw,relatime",
			"Optional": "shared:16",
			"FSType": "hugetlbfs",
			"Source": "hugetlbfs",
			"VFSOptions": "rw,pagesize=2M"
		},
		{
			"ID": 40,
			"Parent": 26,
			"Major": 0,
			"Minor": 20,
			"Root": "/",
			"Mountpoint": "/dev/mqueue",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:17",
			"FSType": "mqueue",
			"Source": "mqueue",
			"VFSOptions": "rw"
		},
		{
			"ID": 41,
			"Parent": 24,
			"Major": 0,
			"Minor": 12,
			"Root": "/",
			"Mountpoint": "/sys/kernel/tracing",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:18",
			"FSType": "tracefs",
			"Source": "tracefs",
			"VFSOptions": "rw"
		},
		{
			"ID": 42,
			"Parent": 24,
			"Major": 0,
			"Minor": 34,
			"Root": "/",
			"Mountpoint": "/sys/fs/fuse/connections",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:19",
			"FSType": "fusectl",
			"Source": "fusectl",
			"VFSOptions": "rw"
		},
		{
			"ID": 43,
			"Parent": 24,
			"Major": 0,
			"Minor": 21,
			"Root": "/",
			"Mountpoint": "/sys/kernel/config",
			"Options": "rw,nosuid,nodev,noexec,relatime",
			"Optional": "shared:20",
			"FSType": "configfs",
			"Source": "configfs",
			"VFSOptions": "rw"
		},
		{
			"ID": 66,
			"Parent": 28,
			"Major": 0,
			"Minor": 35,
			"Root": "/",
			"Mountpoint": "/run/credentials/systemd-sysusers.service",
			"Options": "ro,nosuid,nodev,noexec,relatime",
			"Optional": "shared:21",
			"FSType": "ramfs",
			"Source": "none",
			"VFSOptions": "rw,mode=700"
		},
		{
			"ID": 91,
			"Parent": 29,
			"Major": 8,
			"Minor": 15,
			"Root": "/",
			"Mountpoint": "/boot/efi",
			"Options": "rw,relatime",
			"Optional": "shared:31",
			"FSType": "vfat",
			"Source": "/dev/sda15",
			"VFSOptions": "rw,fmask=0077,dmask=0077,codepage=437,iocharset=iso8859-1,shortname=mixed,errors=remount-ro"
		},
		{
			"ID": 338,
			"Parent": 28,
			"Major": 0,
			"Minor": 45,
			"Root": "/",
			"Mountpoint": "/run/user/1000",
			"Options": "rw,nosuid,nodev,relatime",
			"Optional": "shared:206",
			"FSType": "tmpfs",
			"Source": "tmpfs",
			"VFSOptions": "rw,size=401700k,nr_inodes=100425,mode=700,uid=1000,gid=1000,inode64"
		},
		{
			"ID": 611,
			"Parent": 338,
			"Major": 0,
			"Minor": 46,
			"Root": "/",
			"Mountpoint": "/run/user/1000/gvfs",
			"Options": "rw,nosuid,nodev,relatime",
			"Optional": "shared:327",
			"FSType": "fuse.gvfsd-fuse",
			"Source": "gvfsd-fuse",
			"VFSOptions": "rw,user_id=1000,group_id=1000"
		},
		{
			"ID": 836,
			"Parent": 338,
			"Major": 0,
			"Minor": 51,
			"Root": "/",
			"Mountpoint": "/run/user/1000/doc",
			"Options": "rw,nosuid,nodev,relatime",
			"Optional": "shared:447",
			"FSType": "fuse.portal",
			"Source": "portal",
			"VFSOptions": "rw,user_id=1000,group_id=1000"
		}
	]
	`

	if err := json.Unmarshal([]byte(_allMountInfoJSON), &_allMountInfo); err != nil {
		panic(err)
	}
}

func (m *MountInfoMock) GetMounts(filter mountinfo.FilterFunc) ([]*mountinfo.Info, error) {
	filteredMounts := make([]*mountinfo.Info, 0)

	for i := range _allMountInfo {
		var skip, stop bool
		if filter != nil {
			skip, stop = filter(&_allMountInfo[i])
			if skip {
				continue
			}
		}

		filteredMounts = append(filteredMounts, &_allMountInfo[i])

		if stop {
			break
		}
	}

	return filteredMounts, nil
}

func NewMountInfoMock() wrapper.MountInfoWrapper {
	return &MountInfoMock{}
}

func TestGetMounts(t *testing.T) {
	mountInfoMock := NewMountInfoMock()

	localStorageService := NewLocalStorageService(mountInfoMock)

	mounts, err := localStorageService.GetMounts(codegen.GetMountsParams{})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(mounts), len(_allMountInfo))
}

func TestGetMountsWithFilter(t *testing.T) {
	mountInfoMock := NewMountInfoMock()

	localStorageService := NewLocalStorageService(mountInfoMock)

	expectedMount := _allMountInfo[0]

	// by ID
	expectedMountID := strconv.Itoa(expectedMount.ID)

	mounts, err := localStorageService.GetMounts(codegen.GetMountsParams{
		Id: &expectedMountID,
	})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(mounts), 1)
	assert.Equal(t, *mounts[0].Id, expectedMount.ID)

	// by mount point
	mounts, err = localStorageService.GetMounts(codegen.GetMountsParams{
		MountPoint: &expectedMount.Mountpoint,
	})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(mounts), 1)
	assert.Equal(t, *mounts[0].Mountpoint, expectedMount.Mountpoint)

	// by type
	expectedMountType := "tmpfs"

	expectedMountsByType := make([]*mountinfo.Info, 0)
	for i := range _allMountInfo {
		if _allMountInfo[i].FSType == expectedMountType {
			expectedMountsByType = append(expectedMountsByType, &_allMountInfo[i])
		}
	}

	mounts, err = localStorageService.GetMounts(codegen.GetMountsParams{
		Type: &expectedMountType,
	})

	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(mounts), len(expectedMountsByType))

	for i := range mounts {
		assert.Equal(t, *mounts[i].Id, expectedMountsByType[i].ID)
		assert.Equal(t, *mounts[i].Mountpoint, expectedMountsByType[i].Mountpoint)
		assert.Equal(t, *mounts[i].Type, expectedMountsByType[i].FSType)
		assert.Equal(t, *mounts[i].Source, expectedMountsByType[i].Source)
		assert.Equal(t, *mounts[i].Options, expectedMountsByType[i].Options)
	}
}
