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

	if m.Model != "" {
		name += "_" + m.Model
	}

	return filepath.Join(defaultMountPath, name)
}
