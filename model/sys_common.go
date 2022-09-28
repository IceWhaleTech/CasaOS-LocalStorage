package model

type CommonModel struct {
	RuntimePath string
}

type APPModel struct {
	LogPath     string
	LogSaveName string
	LogFileExt  string
	ShellPath   string
	DBPath      string
}

// 服务配置
type ServerModel struct {
	USBAutoMount   string
	EnableMergerFS string
}
