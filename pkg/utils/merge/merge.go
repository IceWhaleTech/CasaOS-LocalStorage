package merge

import (
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"
)

func IsMergerFSInstalled() bool {
	paths := []string{
		"/sbin/mount.mergerfs", "/usr/sbin/mount.mergerfs", "/usr/local/sbin/mount.mergerfs",
		"/bin/mount.mergerfs", "/usr/bin/mount.mergerfs", "/usr/local/bin/mount.mergerfs",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			logger.Info("mergerfs is installed", zap.String("path", path))
			return true
		}
	}

	logger.Error("mergerfs is not installed at any path", zap.String("paths", strings.Join(paths, ", ")))
	return false
}
