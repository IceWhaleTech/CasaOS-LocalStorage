package common

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	FStabPassDoNotCheck      = 0
	FStabPassCheckDuringBoot = 1
	FStabPassCheckAfterBoot  = 2

	DefaultFStabPath = "/etc/fstab"
)

var (
	ErrInvalidFSTabEntry                     = errors.New("invalid fstab entry")
	ErrDifferentFSTabEntryWithSameMountPoint = errors.New("a different fstab entry with the same mount point already exists")
)

type (
	FSTabEntry struct {
		Source     string
		MountPoint string
		FSType     string
		Options    string
		Dump       int
		Pass       int
	}

	FStab struct {
		fstabPath string
	}
)

func (e *FSTabEntry) String() string {
	return e.Source + "\t" + e.MountPoint + "\t" + e.FSType + "\t" + e.Options + "\t" + strconv.Itoa(e.Dump) + "\t" + strconv.Itoa(e.Pass)
}

func (f *FStab) Add(e FSTabEntry, replace bool) error {
	entry, err := f.GetEntryByMountPoint(e.MountPoint)
	if err != nil {
		return err
	}

	if entry != nil {
		if !replace ||
			entry.Source != e.Source ||
			entry.FSType != e.FSType ||
			entry.Options != e.Options ||
			entry.Dump != e.Dump ||
			entry.Pass != e.Pass {
			return ErrDifferentFSTabEntryWithSameMountPoint
		}

		if err := f.RemoveByMountPoint(e.MountPoint, false); err != nil {
			return err
		}
	}

	if err := copy(f.fstabPath, f.fstabPath+".casaos.bak"); err != nil {
		return err
	}

	fstabFile, err := os.OpenFile(f.fstabPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer fstabFile.Close()

	_, err = fstabFile.WriteString("\n# Added by the CasaOS Local Storage service\n")
	if err != nil {
		return err
	}

	_, err = fstabFile.WriteString(e.String() + "\n")
	if err != nil {
		return err
	}

	_, err = fstabFile.WriteString("\n") // newline
	return err
}

func (f *FStab) RemoveByMountPoint(mountpoint string, comment bool) error {
	FStabPathNew := f.fstabPath + ".casaos.new"
	FStabFileNew, err := os.OpenFile(FStabPathNew, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	if err := foreachLine(f.fstabPath, func(line string) error {
		entry, _ := parseEntry(line)
		if entry != nil && entry.MountPoint == mountpoint {
			if comment {
				_, err := FStabFileNew.WriteString("#" + line + "\n")
				return err
			}
			return nil
		}

		_, err := FStabFileNew.WriteString(line + "\n")
		return err
	}); err != nil {
		return err
	}

	if err := copy(f.fstabPath, f.fstabPath+".casaos.bak"); err != nil {
		return err
	}

	return os.Rename(FStabPathNew, f.fstabPath)
}

func (f *FStab) GetEntries() ([]*FSTabEntry, error) {
	entries := []*FSTabEntry{}

	if err := foreachLine(f.fstabPath, func(line string) error {
		entry, err := parseEntry(line)
		if err != nil {
			return err
		}
		if entry != nil {
			entries = append(entries, entry)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return entries, nil
}

func (f *FStab) GetEntryByMountPoint(mountpoint string) (*FSTabEntry, error) {
	entries, err := f.GetEntries()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.MountPoint == mountpoint {
			return entry, nil
		}
	}

	return nil, nil
}

func GetFSTab() *FStab {
	return &FStab{
		fstabPath: DefaultFStabPath,
	}
}

func parseEntry(line string) (*FSTabEntry, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := strings.Fields(line)
	if len(fields) < 4 || len(fields) > 6 {
		return nil, nil
	}

	entry := FSTabEntry{
		Dump: 0,
		Pass: FStabPassDoNotCheck,
	}

	entry.Source = fields[0]
	entry.MountPoint = fields[1]
	entry.FSType = fields[2]
	entry.Options = fields[3]

	if len(fields) > 4 {
		dump, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, ErrInvalidFSTabEntry
		}
		entry.Dump = dump
	}

	if len(fields) > 5 {
		pass, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, ErrInvalidFSTabEntry
		}
		entry.Pass = pass
	}

	return &entry, nil
}

func foreachLine(path string, handle func(line string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		if err := handle(line); err != nil {
			return err
		}
	}

	return nil
}

func copy(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
