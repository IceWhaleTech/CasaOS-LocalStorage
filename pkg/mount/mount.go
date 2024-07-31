package mount

import "github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/command"

func Mount(source string, mountpoint string, fstype *string, options *string) error {
	args := []string{"--verbose"}

	if fstype != nil && *fstype != "" {
		args = append(args, "-t", *fstype)
	}

	if options != nil && *options != "" {
		args = append(args, "-o", *options)
	}

	args = append(args, source, mountpoint)

	if _, err := command.ExecuteCommand("mount", args...); err != nil {
		return err
	}

	return nil
}

func UmountByMountPoint(mountpoint string) error {
	if _, err := command.ExecuteCommand("umount", "--force", "--verbose", "--quiet", mountpoint); err != nil {
		return err
	}

	return nil
}

func UmountByDevice(device string) error {
	if _, err := command.ExecuteCommand("umount", "--force", "--verbose", "--quiet", "--recursive", device); err != nil {
		return err
	}

	return nil
}
