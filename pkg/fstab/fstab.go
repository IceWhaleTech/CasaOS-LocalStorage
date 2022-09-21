package fstab

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	PassDoNotCheck      = 0
	PassCheckDuringBoot = 1
	PassCheckAfterBoot  = 2

	DefaultPath = "/etc/fstab"
)

var (
	_fstab *FStab

	ErrInvalidFSTabEntry                     = errors.New("invalid fstab entry")
	ErrDifferentFSTabEntryWithSameMountPoint = errors.New("a different fstab entry with the same mount point already exists")
)

type (
	Entry struct {
		// The device name, label, UUID, or other means of specifying the partition or data source this entry refers to.
		Source string

		// Where the contents of the device may be accessed after mounting
		MountPoint string

		// The type of file system to be mounted.
		FSType string

		// Options describing various other aspects of the file system, such as whether it is automatically mounted at boot, which users may mount or access it, whether it may be written to or only read from, its size, and so forth; the special option defaults refers to a pre-determined set of options depending on the file system type.
		Options string

		// A number indicating whether and how often the file system should be backed up by the dump program; a zero indicates the file system will never be automatically backed up.
		Dump int

		// A number indicating the order in which the fsck program will check the devices for errors at boot time
		Pass int
	}

	FStab struct {
		path string
	}
)

func (e *Entry) String() string {
	return e.Source + "\t" + e.MountPoint + "\t" + e.FSType + "\t" + e.Options + "\t" + strconv.Itoa(e.Dump) + "\t" + strconv.Itoa(e.Pass)
}

func (f *FStab) Add(e Entry, replace bool) error {
	entry, err := f.GetEntryByMountPoint(e.MountPoint)
	if err != nil {
		return err
	}

	if entry != nil {
		if !replace &&
			(entry.Source != e.Source ||
				entry.FSType != e.FSType ||
				entry.Options != e.Options ||
				entry.Dump != e.Dump ||
				entry.Pass != e.Pass) {
			return ErrDifferentFSTabEntryWithSameMountPoint
		}

		if err := f.RemoveByMountPoint(e.MountPoint, false); err != nil {
			return err
		}
	}

	if err := copy(f.path, f.path+".casaos.bak"); err != nil {
		return err
	}

	fstabFile, err := os.OpenFile(f.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer fstabFile.Close()

	_, err = fstabFile.WriteString(e.String() + "\t# Added by the CasaOS\n")
	if err != nil {
		return err
	}

	return err
}

func (f *FStab) RemoveByMountPoint(mountpoint string, comment bool) error {
	FStabPathNew := f.path + ".casaos.new"
	FStabFileNew, err := os.OpenFile(FStabPathNew, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	if err := foreachLine(f.path, func(line string) error {
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

	if err := copy(f.path, f.path+".casaos.bak"); err != nil {
		return err
	}

	return os.Rename(FStabPathNew, f.path)
}

func (f *FStab) GetEntries() ([]*Entry, error) {
	entries := []*Entry{}

	if err := foreachLine(f.path, func(line string) error {
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

func (f *FStab) GetEntryByMountPoint(mountpoint string) (*Entry, error) {
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

func Get() *FStab {
	if _fstab == nil {
		_fstab = &FStab{
			path: DefaultPath,
		}
	}

	return _fstab
}

func parseEntry(line string) (*Entry, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil, nil
	}

	entry := Entry{
		Dump: 0,
		Pass: PassDoNotCheck,
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
