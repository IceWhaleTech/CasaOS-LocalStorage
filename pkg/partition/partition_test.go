package partition

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParsePartedOutput(t *testing.T) {
	out := []byte(`
		NR="1" START="2048" END="1050623" SECTORS="1048576" SIZE="536870912" NAME="" UUID="81677ad7-d384-304f-8e72-eecb68f9aa5e" TYPE="0fc63daf-8483-4772-8e79-3d69d8477de4" FLAGS="0x0" SCHEME="gpt"
		NR="2" START="1050624" END="2097118" SECTORS="1046495" SIZE="535805440" NAME="" UUID="d5f58ed1-d0d5-f549-b81e-387f72e9a2d1" TYPE="0fc63daf-8483-4772-8e79-3d69d8477de4" FLAGS="0x0" SCHEME="gpt"
	`)

	partitions := parsePARTXOutput(out)

	if len(partitions) != 2 {
		t.Errorf("Expected 2 partitions, got %d", len(partitions))
	}

	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["NR"], "1")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["START"], "2048")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["END"], "1050623")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["SECTORS"], "1048576")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["SIZE"], "536870912")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["NAME"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["UUID"], "81677ad7-d384-304f-8e72-eecb68f9aa5e")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["TYPE"], "0fc63daf-8483-4772-8e79-3d69d8477de4")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FLAGS"], "0x0")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["SCHEME"], "gpt")

	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["NR"], "2")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["START"], "1050624")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["END"], "2097118")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["SECTORS"], "1046495")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["SIZE"], "535805440")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["NAME"], "")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["UUID"], "d5f58ed1-d0d5-f549-b81e-387f72e9a2d1")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["TYPE"], "0fc63daf-8483-4772-8e79-3d69d8477de4")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["FLAGS"], "0x0")
	assert.Equal(t, partitions["d5f58ed1-d0d5-f549-b81e-387f72e9a2d1"]["SCHEME"], "gpt")
}

func TestParseLSBLKOutput(t *testing.T) {
	out := []byte(`
		NAME="sdb" KNAME="sdb" PATH="/dev/sdb" MAJ:MIN="8:16" FSAVAIL="" FSSIZE="" FSTYPE="" FSUSED="" FSUSE%="" FSROOTS="" FSVER="" MOUNTPOINT="" MOUNTPOINTS="" LABEL="" UUID="" PTUUID="682acbc1-deb1-4142-aae2-9bc381c750d7" PTTYPE="gpt" PARTTYPE="" PARTTYPENAME="" PARTLABEL="" PARTUUID="" PARTFLAGS="" RA="128" RO="0" RM="0" HOTPLUG="0" MODEL="Virtual Disk" SERIAL="60022480fa6b8ed470ca1574f4109e87" SIZE="1073741824" STATE="running" OWNER="root" GROUP="disk" MODE="brw-rw----" ALIGNMENT="0" MIN-IO="4096" OPT-IO="0" PHY-SEC="4096" LOG-SEC="512" ROTA="1" SCHED="none" RQ-SIZE="316" TYPE="disk" DISC-ALN="0" DISC-GRAN="2097152" DISC-MAX="4294966784" DISC-ZERO="0" WSAME="0" WWN="0x60022480fa6b8ed470ca1574f4109e87" RAND="1" PKNAME="" HCTL="0:0:0:1" TRAN="" SUBSYSTEMS="block:scsi:vmbus:acpi" REV="1.0 " VENDOR="Msft    " ZONED="none" DAX="0"
		NAME="sdb1" KNAME="sdb1" PATH="/dev/sdb1" MAJ:MIN="8:17" FSAVAIL="" FSSIZE="" FSTYPE="" FSUSED="" FSUSE%="" FSROOTS="" FSVER="" MOUNTPOINT="" MOUNTPOINTS="" LABEL="" UUID="" PTUUID="682acbc1-deb1-4142-aae2-9bc381c750d7" PTTYPE="gpt" PARTTYPE="0fc63daf-8483-4772-8e79-3d69d8477de4" PARTTYPENAME="Linux filesystem" PARTLABEL="" PARTUUID="81677ad7-d384-304f-8e72-eecb68f9aa5e" PARTFLAGS="" RA="128" RO="0" RM="0" HOTPLUG="0" MODEL="" SERIAL="" SIZE="536870912" STATE="" OWNER="root" GROUP="disk" MODE="brw-rw----" ALIGNMENT="0" MIN-IO="4096" OPT-IO="0" PHY-SEC="4096" LOG-SEC="512" ROTA="1" SCHED="none" RQ-SIZE="316" TYPE="part" DISC-ALN="1048576" DISC-GRAN="2097152" DISC-MAX="4294966784" DISC-ZERO="0" WSAME="0" WWN="0x60022480fa6b8ed470ca1574f4109e87" RAND="1" PKNAME="sdb" HCTL="" TRAN="" SUBSYSTEMS="block:scsi:vmbus:acpi" REV="" VENDOR="" ZONED="none" DAX="0"
	`)

	partitions := parseLSBLKOutput(out)
	assert.Equal(t, len(partitions), 1)

	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["NAME"], "sdb1")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["KNAME"], "sdb1")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["PATH"], "/dev/sdb1")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["MAJ:MIN"], "8:17")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSAVAIL"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSSIZE"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSTYPE"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSUSED"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSUSE%"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSROOTS"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["FSVER"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["MOUNTPOINT"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["MOUNTPOINTS"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["LABEL"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["UUID"], "")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["PTUUID"], "682acbc1-deb1-4142-aae2-9bc381c750d7")
	assert.Equal(t, partitions["81677ad7-d384-304f-8e72-eecb68f9aa5e"]["PTTYPE"], "gpt")
}

func TestMergeOutputs(t *testing.T) {
	partxOut := []byte(`
		NR="1" START="2048" END="1050623" SECTORS="1048576" SIZE="536870912" NAME="" UUID="81677ad7-d384-304f-8e72-eecb68f9aa5e" TYPE="0fc63daf-8483-4772-8e79-3d69d8477de4" FLAGS="0x0" SCHEME="gpt"
		NR="2" START="1050624" END="2097118" SECTORS="1046495" SIZE="535805440" NAME="" UUID="d5f58ed1-d0d5-f549-b81e-387f72e9a2d1" TYPE="0fc63daf-8483-4772-8e79-3d69d8477de4" FLAGS="0x0" SCHEME="gpt"
	`)

	lsblkOut := []byte(`
		NAME="sdb" KNAME="sdb" PATH="/dev/sdb" MAJ:MIN="8:16" FSAVAIL="" FSSIZE="" FSTYPE="" FSUSED="" FSUSE%="" FSROOTS="" FSVER="" MOUNTPOINT="" MOUNTPOINTS="" LABEL="" UUID="" PTUUID="682acbc1-deb1-4142-aae2-9bc381c750d7" PTTYPE="gpt" PARTTYPE="" PARTTYPENAME="" PARTLABEL="" PARTUUID="" PARTFLAGS="" RA="128" RO="0" RM="0" HOTPLUG="0" MODEL="Virtual Disk" SERIAL="60022480fa6b8ed470ca1574f4109e87" SIZE="1073741824" STATE="running" OWNER="root" GROUP="disk" MODE="brw-rw----" ALIGNMENT="0" MIN-IO="4096" OPT-IO="0" PHY-SEC="4096" LOG-SEC="512" ROTA="1" SCHED="none" RQ-SIZE="316" TYPE="disk" DISC-ALN="0" DISC-GRAN="2097152" DISC-MAX="4294966784" DISC-ZERO="0" WSAME="0" WWN="0x60022480fa6b8ed470ca1574f4109e87" RAND="1" PKNAME="" HCTL="0:0:0:1" TRAN="" SUBSYSTEMS="block:scsi:vmbus:acpi" REV="1.0 " VENDOR="Msft    " ZONED="none" DAX="0"
		NAME="sdb1" KNAME="sdb1" PATH="/dev/sdb1" MAJ:MIN="8:17" FSAVAIL="" FSSIZE="" FSTYPE="" FSUSED="" FSUSE%="" FSROOTS="" FSVER="" MOUNTPOINT="" MOUNTPOINTS="" LABEL="" UUID="" PTUUID="682acbc1-deb1-4142-aae2-9bc381c750d7" PTTYPE="gpt" PARTTYPE="0fc63daf-8483-4772-8e79-3d69d8477de4" PARTTYPENAME="Linux filesystem" PARTLABEL="" PARTUUID="81677ad7-d384-304f-8e72-eecb68f9aa5e" PARTFLAGS="" RA="128" RO="0" RM="0" HOTPLUG="0" MODEL="" SERIAL="" SIZE="536870912" STATE="" OWNER="root" GROUP="disk" MODE="brw-rw----" ALIGNMENT="0" MIN-IO="4096" OPT-IO="0" PHY-SEC="4096" LOG-SEC="512" ROTA="1" SCHED="none" RQ-SIZE="316" TYPE="part" DISC-ALN="1048576" DISC-GRAN="2097152" DISC-MAX="4294966784" DISC-ZERO="0" WSAME="0" WWN="0x60022480fa6b8ed470ca1574f4109e87" RAND="1" PKNAME="sdb" HCTL="" TRAN="" SUBSYSTEMS="block:scsi:vmbus:acpi" REV="" VENDOR="" ZONED="none" DAX="0"
	`)

	partxPartitions := parsePARTXOutput(partxOut)
	lsblkPartitions := parseLSBLKOutput(lsblkOut)

	partitions := mergeOutputs(lsblkPartitions, partxPartitions)
	assert.Equal(t, len(partitions), 1)

	partition := partitions[0]

	assert.Equal(t, partition.LSBLKProperties["PARTUUID"], partition.PARTXProperties["UUID"])
}
