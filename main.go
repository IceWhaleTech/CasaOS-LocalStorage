//go:generate bash -c "mkdir -p codegen && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest -package codegen api/local_storage/openapi.yaml > codegen/local_storage_api.go"

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

	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	util_http "github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	gateway_common "github.com/IceWhaleTech/CasaOS-Gateway/common"
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
	"github.com/robfig/cron"
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
		ensureDefaultMergePoint()
		service.MyService.LocalStorage().CheckMergeMount()
	}

	ensureDefaultDirectories()
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

func ensureDefaultMergePoint() {
	mountPoint := "/DATA"
	sourceBasePath := constants.DefaultFilePath

	logger.Info("ensure default merge point exists", zap.String("mount point", mountPoint), zap.String("sourceBasePath", sourceBasePath))

	existingMerges, err := service.MyService.LocalStorage().GetMergeAll(&mountPoint)
	if err != nil {
		panic(err)
	}

	// check if /DATA is already a merge point
	if len(existingMerges) > 0 {
		if len(existingMerges) > 1 {
			logger.Error("more than one merge point with the same mount point found", zap.String("mount point", mountPoint))
		}
		return
	}

	if _, err := service.MyService.LocalStorage().SetMerge(&model2.Merge{
		FSType:         fs.MergerFSFullName,
		MountPoint:     mountPoint,
		SourceBasePath: &sourceBasePath,
	}); err != nil {
		if errors.Is(err, v2.ErrMergeMountPointAlreadyExists) {
			logger.Info(err.Error(), zap.String("mount point", mountPoint))
		} else if errors.Is(err, v2.ErrMountPointIsNotEmpty) {
			logger.Error("Mount point "+mountPoint+" is not empty - disabling MergerFS", zap.String("mount point", mountPoint))
			config.ServerInfo.EnableMergerFS = "False"
		} else {
			panic(err)
		}
	}
}

func main() {
	go monitorUSB()

	sendStorageStats()

	crontab := cron.New()
	if err := crontab.AddFunc("*/5 * * * * *", func() { sendStorageStats() }); err != nil {
		logger.Error("crontab add func error", zap.Error(err))
	}

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
		err = service.MyService.Gateway().CreateRoute(&gateway_common.Route{
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
