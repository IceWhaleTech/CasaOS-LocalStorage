package mount

import (
	"errors"
	"os/exec"
)

func Mount(source string, mountpoint string, fstype *string, options *string) error {
	args := []string{"--verbose"}

	if fstype != nil && *fstype != "" {
		args = append(args, "-t", *fstype)
	}

	if options != nil && *options != "" {
		args = append(args, "-o", *options)
	}

	args = append(args, source, mountpoint)

	if _, err := executeCommand("mount", args...); err != nil {
		return err
	}

	return nil
}

func UmountByMountPoint(mountpoint string) error {
	if _, err := executeCommand("umount", "--force", "--verbose", "--quiet", mountpoint); err != nil {
		return err
	}

	return nil
}

func UmountByDevice(device string) error {
	if _, err := executeCommand("umount", "--force", "--verbose", "--quiet", "--recursive", device); err != nil {
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
