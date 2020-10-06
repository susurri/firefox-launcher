package launcher

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/kyoh86/xdg"
)

func openpid() int {
	lockfile := filepath.Join(xdg.RuntimeDir(), "firefox_launcher.pid")
	fd, err := syscall.Open(lockfile, syscall.O_CREAT|syscall.O_WRONLY, 0600)
	if err != nil { log.Fatal(err) }
	return fd
}

func lockpid(fd int) int {
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		syscall.Close(fd)
	  if err != nil { log.Fatal(err) }
	}
	syscall.Ftruncate(fd, 0)
	var buf []byte
	syscall.Write(fd, strconv.AppendInt(buf, int64(os.Getpid()), 10))
	return fd
}
