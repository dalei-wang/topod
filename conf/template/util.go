package template

import (
	"errors"
	"io"
	"os"
	"path"
)

func appendPrefixKeys(prefix string, keys []string) []string {
	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = path.Join(prefix, key)
	}
	return result
}

func isFileExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
