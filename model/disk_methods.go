package model

import "path/filepath"

const defaultMountPath = "/mnt"

func (m *LSBLKModel) GetMountPoint(name string) string {
	if name == "" {
		name = "Storage_" + m.Name
	}

	if m.Label != "" {
		name += "_" + m.Label
	}

	return filepath.Join(defaultMountPath, name)
}
