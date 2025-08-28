package version

import (
	"fmt"
	"runtime"
)

var (
	// 这些变量在构建时通过ldflags设置
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// GetVersion 返回版本信息
func GetVersion() string {
	return fmt.Sprintf("singgen %s", Version)
}

// GetFullVersion 返回完整的版本信息
func GetFullVersion() string {
	return fmt.Sprintf(`singgen %s
Git Commit: %s
Build Time: %s
Go Version: %s
OS/Arch: %s/%s`,
		Version,
		GitCommit,
		BuildTime,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}