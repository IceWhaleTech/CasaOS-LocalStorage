package partition

import (
	"bytes"
	"errors"
	"os/exec"
	"strconv"
)

type Partition struct {
	LSBLKProperties map[string]string
	PARTXProperties map[string]string
}

// rootDevice - root device, e.g. /dev/sda
func GetPartitions(rootDevice string) ([]Partition, error) {
	// partx
	out, err := executeCommand("partx", "--pairs", "--bytes", "--output-all", rootDevice)
	if err != nil {
		return nil, err
	}
	partxPartitions := parsePARTXOutput(out)

	// lsblk
	out, err = executeCommand("lsblk", "--pairs", "--bytes", "--output-all", rootDevice)
	if err != nil {
		return nil, err
	}
	lsblkPartitions := parseLSBLKOutput(out)

	// merge
	partitions := mergeOutputs(lsblkPartitions, partxPartitions)

	return partitions, nil
}

// rootDevice - root device, e.g. /dev/sda
func AddPartition(rootDevice string) error {
	// add partition
	if _, err := executeCommand("parted", "-s", rootDevice, "mkpart", "primary", "ext4", "0", "100%"); err != nil {
		return err
	}

	// inform the operating system about partition table changes
	if _, err := executeCommand("partprobe", "-s", rootDevice); err != nil {
		return err
	}

	return nil
}

func CreatePartitionTable(rootDevice string) error {
	// create partition table
	if _, err := executeCommand("parted", "-s", rootDevice, "mklabel", "gpt"); err != nil {
		return err
	}
	return nil
}

// partitionDevice - partition device, e.g. /dev/sda1
func FormatPartition(partitionDevice string) error {
	if _, err := executeCommand(
		"mkfs.ext4",
		"-v",      // Verbose execution.
		"-m", "1", // Specify  the  percentage of the file system blocks reserved for the super-user.
		"-F",
		partitionDevice,
	); err != nil {
		return err
	}

	return nil
}

// rootDevice - root device, e.g. /dev/sda
//
// number - partition number, e.g. 1
func DeletePartition(rootDevice string, number int) error {
	n := strconv.Itoa(number)

	// delete partition
	if _, err := executeCommand("sfdisk", "--delete", rootDevice, n); err != nil {
		return err
	}

	// inform the operating system about partition table changes
	if _, err := executeCommand("partprobe", "-s", rootDevice); err != nil {
		return err
	}

	return nil
}

func executeCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	println(cmd.String())

	out, err := cmd.Output()
	println(string(out))
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			message := string(exitError.Stderr)
			println(message)
			return nil, errors.New(message)
		}
		return nil, err
	}

	return out, nil
}

func parsePARTXOutput(out []byte) map[string]map[string]string {
	partitions := map[string]map[string]string{}
	for _, buf := range bytes.Split(out, []byte("\n")) {
		if len(buf) == 0 {
			continue
		}

		partition := parsePairs(buf)
		if partition["UUID"] == "" {
			continue
		}

		partitions[partition["UUID"]] = partition
	}
	return partitions
}

func parseLSBLKOutput(out []byte) map[string]map[string]string {
	partitions := map[string]map[string]string{}
	for _, buf := range bytes.Split(out, []byte("\n")) {
		if len(buf) == 0 {
			continue
		}

		partition := parsePairs(buf)
		if partition["PARTUUID"] == "" {
			continue
		}

		partitions[partition["PARTUUID"]] = partition
	}
	return partitions
}

func mergeOutputs(lsblkPartitions, partxPartitions map[string]map[string]string) []Partition {
	partitions := []Partition{}
	for uuid, partxPartition := range partxPartitions {
		lsblkPartition, ok := lsblkPartitions[uuid]
		if !ok {
			continue
		}
		partitions = append(partitions, Partition{
			LSBLKProperties: lsblkPartition,
			PARTXProperties: partxPartition,
		})
	}

	return partitions
}

func parsePairs(buf []byte) map[string]string {
	pairs := map[string]string{}
	for _, field := range bytes.Fields(buf) {
		kv := bytes.Split(field, []byte("="))
		if len(kv) != 2 {
			continue
		}
		pairs[string(kv[0])] = string(bytes.Trim(kv[1], "\""))
	}

	return pairs
}
