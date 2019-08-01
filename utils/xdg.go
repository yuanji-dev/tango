package utils

import (
	"os"
	"path/filepath"
)

func GetDataDir(name string) string {
	var basedir string
	if env := os.Getenv("XDG_DATA_HOME"); env != "" {
		basedir = env
	} else {
		basedir = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(basedir, name)
}
