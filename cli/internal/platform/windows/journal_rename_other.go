//go:build !windows

package windows

import "os"

func replaceJournalFile(source, destination string) error {
	return os.Rename(source, destination)
}
