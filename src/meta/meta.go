package meta

import (
	"fmt"
	"runtime/debug"
	goruntime "runtime"
	"path/filepath"
	"os"
)

var (
	VERSION    = "2.0.0 Beta"
	GOVER      = func () string {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return "unknown"
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.modified" && setting.Value == "true" {
				return fmt.Sprintf("%s (modified)", goruntime.Version())
			}
		}
		return goruntime.Version()
	}()
	FULLVER    = fmt.Sprintf("version %s, %s %s", VERSION, GOVER, COMPILER)
	OS         = goruntime.GOOS
	ARCH       = goruntime.GOARCH
	COMPILER   = goruntime.Compiler
	ENVPATH    = func() string {
		userDir, err := os.UserHomeDir()
		if err != nil {
			return "unknown"
		}
		envRelPath := []string{"MeowPlusPlus", "env"}
		var res string
		switch OS {
		case "darwin":
			res = filepath.Join(append([]string{userDir, "Library", "Application Support"}, envRelPath...)...)
		case "linux":
			res = filepath.Join(append([]string{userDir, ".config"}, envRelPath...)...)
		case "windows":
			res = filepath.Join(append([]string{userDir, "AppData", "Roaming"}, envRelPath...)...)
		}
		if res == "" {
			return "unknown"
		}
		if _, err := os.Stat(res); os.IsNotExist(err) {
			err := os.MkdirAll(res, 0755)
			if err != nil {
				return "unknown"
			}
		}
		return res
	}()
	SOURCEPATH = func() string {
		srcPath := filepath.Join(ENVPATH, "src")
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			err := os.MkdirAll(srcPath, 0755)
			if err != nil {
				return "unknown"
			}
		}
		return srcPath
	}()
)

func GetEnvFile(relPath ...string) string {
	return filepath.Join(append([]string{ENVPATH}, relPath...)...)
}
