// +build darwin dragonfly freebsd linux netbsd openbsd

package huobi

import (
	"os"
	"path/filepath"
	"syscall"
)

type FileLock struct {
	f *os.File
}

func NewFileLock(fpath string) (fl *FileLock, err error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(fpath), 0777)
		if err != nil {
			return nil, err
		}
		f, err = os.OpenFile(fpath, os.O_RDWR|os.O_CREATE, 0644)
	}
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &FileLock{f: f}, nil
}

func (fl *FileLock) Release() error {
	if err := syscall.Flock(int(fl.f.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}
	return fl.f.Close()
}
