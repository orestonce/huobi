package huobi

import (
	"syscall"
)

type FileLock struct {
	fd syscall.Handle
}

func NewFileLock(path string) (fl *FileLock, err error) {
	fpath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	var access uint32
	access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	fd, err := syscall.CreateFile(fpath, access, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return &FileLock{fd: fd}, nil
}

func (fl *FileLock) Release() error {
	return syscall.Close(fl.fd)
}
