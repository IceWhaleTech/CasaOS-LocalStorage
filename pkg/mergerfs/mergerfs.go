package mergerfs

import (
	"bytes"
	"path/filepath"
	"strings"
	"syscall"
)

func ControlFile(path string) string {
	return filepath.Join(path, ".mergerfs")
}

func ListValues(fspath string) (map[string]string, error) {
	ctrlfile := ControlFile(fspath)

	buf := make([]byte, 4096)
	size, err := syscall.Listxattr(ctrlfile, buf)
	if err != nil {
		return nil, err
	}

	buf = buf[:size]

	values := make(map[string]string)
	for _, keyBuf := range bytes.Split(buf, []byte{0}) {
		if len(keyBuf) == 0 {
			continue
		}
		key := string(keyBuf)
		value := make([]byte, 512)
		size, err := syscall.Getxattr(ctrlfile, key, value)
		if err != nil {
			return nil, err
		}
		value = value[:size]
		values[key] = string(value)
	}

	return values, nil
}

func SetSource(ctrlfile string, sources []string) error {
	key := "user.mergerfs.srcmounts"
	value := []byte(strings.Join(sources, ":"))

	return syscall.Setxattr(ctrlfile, key, value, 0)
}

func GetSource(ctrlfile string) ([]string, error) {
	values, err := ListValues(ctrlfile)
	if err != nil {
		return nil, err
	}

	return strings.Split(values["user.mergerfs.srcmounts"], ":"), nil
}

func AddSource(ctrlfile string, source string) error {
	key := "user.mergerfs.srcmounts"
	value := []byte("+" + source)

	return syscall.Setxattr(ctrlfile, key, value, 0)
}

func RemoveSource(ctrlfile string, source string) error {
	key := "user.mergerfs.srcmounts"
	value := []byte("-" + source)

	return syscall.Setxattr(ctrlfile, key, value, 0)
}

func AddPath(fspath string, path string) error {
	ctrlfile := ControlFile(fspath)
	return AddSource(ctrlfile, path)
}

func RemovePath(fspath string, path string) error {
	ctrlfile := ControlFile(fspath)
	return RemoveSource(ctrlfile, path)
}
