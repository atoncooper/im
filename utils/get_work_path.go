package utils

import (
	"os"
	"path/filepath"
)

func GetWorkPath() (string, error) {

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		cfgPath := filepath.Join(dir, "gateway.yaml")

		if _, err := os.Stat(cfgPath); err == nil {

			return dir, err
		}
		parent := filepath.Dir(dir)

		if parent == dir {
			return "", err
		}
		dir = parent

	}

}
