package mergerfs

import (
	"bytes"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"
)

func ControlFile(fspath string) string {
	return filepath.Join(fspath, ".mergerfs")
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

func SetSource(fspath string, sources []string) error {
	ctrlfile := ControlFile(fspath)

	key := "user.mergerfs.branches"

	sourceMap := make(map[string]interface{})
	for _, source := range sources {
		sourceMap[source] = true
	}

	dedupedSources := make([]string, 0)
	for source := range sourceMap {
		dedupedSources = append(dedupedSources, source)
	}

	value := []byte(strings.Join(dedupedSources, ":"))
	//str, err := command.ExecResultStr("setfattr -n " + key + " -v " + string(string(value)) + " " + ctrlfile)
	err := syscall.Setxattr(ctrlfile, key, value, 0)
	//logger.Error("SetSourceStr", zap.String("str", str))
	if err != nil {
		logger.Error("SetSource", zap.Error(err))
		return err
	}
	return err
}

func GetSource(fspath string) ([]string, error) {
	values, err := ListValues(fspath)
	if err != nil {
		return nil, err
	}

	return strings.Split(values["user.mergerfs.srcmounts"], ":"), nil
}

func AddSource(fspath string, source string) error {
	ctrlfile := ControlFile(fspath)

	key := "user.mergerfs.branches"
	value := []byte("+" + source)

	return syscall.Setxattr(ctrlfile, key, value, 0)
}

func RemoveSource(fspath string, source string) error {
	ctrlfile := ControlFile(fspath)

	key := "user.mergerfs.branches"
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
