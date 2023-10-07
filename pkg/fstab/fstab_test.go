package fstab

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

const fstabContent = `
	# UNCONFIGURED FSTAB FOR BASE SYSTEM
	LABEL=UEFI      /boot/efi       vfat    umask=0077      0 1
	/mnt/sdb:/mnt/sdc       /media  mergerfs        defaults,allow_other,category.create=mfs,moveonenospc=true,minfreespace=1M 0 0
	LABEL=desktop-rootfs    /               ext4    defaults        0 1
`

func TestFSTab(t *testing.T) {
	fstab := &FStab{path: "/tmp/fstab"}

	err := os.WriteFile(fstab.path, []byte(fstabContent), 0o600)
	assert.NilError(t, err)

	entries, err := fstab.GetEntries()
	assert.NilError(t, err)

	assert.Equal(t, len(entries), 3)

	entry, err := fstab.GetEntryByMountPoint("/media")
	assert.NilError(t, err)

	assert.Equal(t, entry.Source, "/mnt/sdb:/mnt/sdc")
	assert.Equal(t, entry.MountPoint, "/media")
	assert.Equal(t, entry.FSType, "mergerfs")
	assert.Equal(t, entry.Options, "defaults,allow_other,category.create=mfs,moveonenospc=true,minfreespace=1M")
	assert.Equal(t, entry.Dump, 0)
	assert.Equal(t, entry.Pass, PassDoNotCheck)

	err = fstab.RemoveByMountPoint(entry.MountPoint, false)
	assert.NilError(t, err)

	nonExistingEntry, err := fstab.GetEntryByMountPoint(entry.MountPoint)
	assert.NilError(t, err)
	assert.Equal(t, nonExistingEntry, (*Entry)(nil))

	err = fstab.Add(*entry, true)
	assert.NilError(t, err)

	entry, err = fstab.GetEntryByMountPoint(entry.MountPoint)
	assert.NilError(t, err)

	assert.Equal(t, entry.Source, "/mnt/sdb:/mnt/sdc")
	assert.Equal(t, entry.MountPoint, "/media")
	assert.Equal(t, entry.FSType, "mergerfs")
	assert.Equal(t, entry.Options, "defaults,allow_other,category.create=mfs,moveonenospc=true,minfreespace=1M")
	assert.Equal(t, entry.Dump, 0)
	assert.Equal(t, entry.Pass, PassDoNotCheck)
}
