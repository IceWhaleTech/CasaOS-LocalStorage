//go:generate bash -c "mkdir -p codegen && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.2 -package codegen api/local_storage/openapi.yaml > codegen/local_storage_api.go"

package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	util_http "github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/cache"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/route"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/fs"
	"github.com/coreos/go-systemd/daemon"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const localhost = "127.0.0.1"

var (
	//go:embed api/index.html
	_docHTML string

	//go:embed api/local_storage/openapi.yaml
	_docYAML string
)

func init() {
	configFlag := flag.String("c", "", "config address")
	dbFlag := flag.String("db", "", "db path")

	versionFlag := flag.Bool("v", false, "version")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.Version)
		os.Exit(0)
	}

	config.InitSetup(*configFlag)

	logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

	if len(*dbFlag) == 0 {
		*dbFlag = config.AppInfo.DBPath
	}

	sqliteDB := sqlite.GetGlobalDB(*dbFlag)

	service.MyService = service.NewService(sqliteDB)

	service.Cache = cache.Init()

	service.MyService.Disk().CheckSerialDiskMount()

	if strings.ToLower(config.ServerInfo.EnableMergerFS) == "true" {
		if !isMergerFSInstalled() {
			config.ServerInfo.EnableMergerFS = "false"
			logger.Info("mergerfs is disabled")
		}
	}

	if strings.ToLower(config.ServerInfo.EnableMergerFS) == "true" {
		if !ensureDefaultMergePoint() {
			config.ServerInfo.EnableMergerFS = "false"
			logger.Info("mergerfs is disabled")
		}
	}

	if strings.ToLower(config.ServerInfo.EnableMergerFS) == "true" {
		service.MyService.LocalStorage().CheckMergeMount()
	}

	checkToken2_11()

	ensureDefaultDirectories()
}

func checkToken2_11() {
	deviceTree, err := service.MyService.USB().GetDeviceTree()
	if err != nil {
		panic(err)
	}

	if service.MyService.USB().GetSysInfo().KernelArch == "aarch64" && strings.ToLower(config.ServerInfo.USBAutoMount) != "true" && strings.Contains(deviceTree, "Raspberry Pi") {
		service.MyService.USB().UpdateUSBAutoMount("False")
		service.MyService.USB().ExecUSBAutoMountShell("False")
	}
}

func ensureDefaultDirectories() {
	sysType := runtime.GOOS
	var dirArray []string
	if sysType == "linux" {
		dirArray = []string{"/DATA/AppData", "/DATA/Documents", "/DATA/Downloads", "/DATA/Gallery", "/DATA/Media/Movies", "/DATA/Media/TV Shows", "/DATA/Media/Music"}
	}

	if sysType == "windows" {
		dirArray = []string{"C:\\CasaOS\\DATA\\AppData", "C:\\CasaOS\\DATA\\Documents", "C:\\CasaOS\\DATA\\Downloads", "C:\\CasaOS\\DATA\\Gallery", "C:\\CasaOS\\DATA\\Media/Movies", "C:\\CasaOS\\DATA\\Media\\TV Shows", "C:\\CasaOS\\DATA\\Media\\Music"}
	}

	if sysType == "darwin" {
		dirArray = []string{"./CasaOS/DATA/AppData", "./CasaOS/DATA/Documents", "./CasaOS/DATA/Downloads", "./CasaOS/DATA/Gallery", "./CasaOS/DATA/Media/Movies", "./CasaOS/DATA/Media/TV Shows", "./CasaOS/DATA/Media/Music"}
	}

	for _, v := range dirArray {
		if err := file.IsNotExistMkDir(v); err != nil {
			logger.Error("ensureDefaultDirectories", zap.Error(err))
		}
	}
}

func isMergerFSInstalled() bool {
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

func ensureDefaultMergePoint() bool {
	mountPoint := "/DATA"
	sourceBasePath := constants.DefaultFilePath

	logger.Info("ensure default merge point exists", zap.String("mount point", mountPoint), zap.String("sourceBasePath", sourceBasePath))

	existingMerges, err := service.MyService.LocalStorage().GetMergeAllFromDB(&mountPoint)
	if err != nil {
		panic(err)
	}

	// check if /DATA is already a merge point
	if len(existingMerges) > 0 {
		if len(existingMerges) > 1 {
			logger.Error("more than one merge point with the same mount point found", zap.String("mount point", mountPoint))
		}
		return true
	}

	merge := &model2.Merge{
		FSType:         fs.MergerFSFullName,
		MountPoint:     mountPoint,
		SourceBasePath: &sourceBasePath,
	}

	if err := service.MyService.LocalStorage().CreateMerge(merge); err != nil {
		if errors.Is(err, v2.ErrMergeMountPointAlreadyExists) {
			logger.Info(err.Error(), zap.String("mount point", mountPoint))
		} else if errors.Is(err, v2.ErrMountPointIsNotEmpty) {
			logger.Error("Mount point "+mountPoint+" is not empty", zap.String("mount point", mountPoint))
			return false
		} else {
			panic(err)
		}
	}

	if err := service.MyService.LocalStorage().CreateMergeInDB(merge); err != nil {
		panic(err)
	}

	return true
}

func main() {
	go monitorUSB()

	sendStorageStats()

	crontab := cron.New(cron.WithSeconds())
	if _, err := crontab.AddFunc("@every 5s", sendStorageStats); err != nil {
		logger.Error("crontab add func error", zap.Error(err))
	}

	crontab.Start()

	defer crontab.Stop()

	listener, err := net.Listen("tcp", net.JoinHostPort(localhost, "0"))
	if err != nil {
		panic(err)
	}

	apiPaths := []string{
		"/v1/usb",
		"/v1/disks",
		"/v1/storage",
		route.V2APIPath,
		route.V2DocPath,
	}
	for _, apiPath := range apiPaths {
		err = service.MyService.Gateway().CreateRoute(&model.Route{
			Path:   apiPath,
			Target: "http://" + listener.Addr().String(),
		})

		if err != nil {
			panic(err)
		}
	}

	v1Router := route.InitV1Router()
	v2Router := route.InitV2Router()
	v2DocRouter := route.InitV2DocRouter(_docHTML, _docYAML)

	mux := &util_http.HandlerMultiplexer{
		HandlerMap: map[string]http.Handler{
			"v1":  v1Router,
			"v2":  v2Router,
			"doc": v2DocRouter,
		},
	}

	if supported, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		logger.Error("Failed to notify systemd that local storage service is ready", zap.Error(err))
	} else if supported {
		logger.Info("Notified systemd that local storage service is ready")
	} else {
		logger.Info("This process is not running as a systemd service.")
	}

	logger.Info("LocalStorage service is listening...", zap.Any("address", listener.Addr().String()))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	err = server.Serve(listener)
	if err != nil {
		panic(err)
	}
}
