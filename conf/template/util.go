package template

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wlsailor/topod/logger"
)

type fileInfo struct {
	Uid  uint32
	Gid  uint32
	Mode os.FileMode
	Md5  string
}

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

func fileStat(name string) (fi fileInfo, err error) {
	if isFileExist(name) {
		f, err := os.Open(name)
		if err != nil {
			return fi, err
		}
		defer f.Close()
		stats, err := f.Stat()
		if err != nil {
			return fi, err
		}
		sys, ok := stats.Sys().(*syscall.Stat_t)
		if ok {
			fi.Uid = sys.Uid
			fi.Gid = sys.Gid
		} else {
			return fi, errors.New("Bad file stats info")
		}

		fi.Mode = stats.Mode()
		h := md5.New()
		io.Copy(h, f)
		fi.Md5 = fmt.Sprintf("%x", h.Sum(nil))
		return fi, nil
	} else {
		return fi, errors.New("File not found")
	}
}

func isSameFile(src, dest string) (bool, error) {
	d, err := fileStat(dest)
	if err != nil {
		return false, err
	}
	s, err := fileStat(src)
	if err != nil {
		return false, err
	}
	r := true
	if s.Uid != d.Uid {
		logger.Log.Info("%s has UID %d which should be %d", dest, d.Uid, s.Uid)
		r = false
	}
	if s.Gid != d.Gid {
		logger.Log.Info("%s has GID %d which should be %d", dest, d.Gid, s.Gid)
		r = false
	}
	if s.Mode != d.Mode {
		logger.Log.Info("%s has mode %s which should be %s", dest, d.Mode, s.Mode)
		r = false
	}
	if s.Md5 != d.Md5 {
		logger.Log.Info("%s has md5 %s which should be %s", dest, d.Md5, s.Md5)
		r = false
	}
	return r, nil
}

func backupFile(src, backupDir string) (string, error) {
	filename := filepath.Base(src)
	backup := filepath.Join(backupDir, filename+time.Now().Format(time.RFC3339Nano))
	err := os.Rename(src, backup)
	if err != nil {
		return "", err
	} else {
		return backup, err
	}
}
