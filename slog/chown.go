//go:build !linux

package _slog

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
