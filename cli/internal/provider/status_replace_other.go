//go:build !windows

package provider

import "os"

func replaceStatusFile(source, destination string) error { return os.Rename(source, destination) }
