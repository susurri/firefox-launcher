package launcher

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/kyoh86/xdg"
	"github.com/shirou/gopsutil/process"
)

func openpid() int {
	lockfile := filepath.Join(xdg.RuntimeDir(), "firefox_launcher.pid")
	fd, err := syscall.Open(lockfile, syscall.O_CREAT|syscall.O_RDWR, 0600)
	if err != nil { log.Fatal(err) }
	return fd
}

func isRunning(fd int) bool {
	buf := make([]byte, 255)
	n, err := syscall.Read(fd, buf)
	if err != nil { log.Fatal(err) }
	if n == 0 {
		return false
	}
	pid, err := strconv.Atoi(string(buf[:n]))
	if err != nil { log.Fatal(err) }
	exist, err := process.PidExists(int32(pid))
	if err != nil { log.Fatal(err) }
	return exist
}

func writepid() {
	fd := openpid()
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		syscall.Close(fd)
	  if err != nil { log.Fatal(err) }
	}
	defer syscall.Close(fd)
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if isRunning(fd) {
		log.Fatal("Another launcher is running")
	}
	err := syscall.Ftruncate(fd, 0)
	if err != nil { log.Fatal(err) }
	_, err = syscall.Seek(fd, 0, os.SEEK_SET)
	if err != nil { log.Fatal(err) }
	var buf []byte
	syscall.Write(fd, strconv.AppendInt(buf, int64(os.Getpid()), 10))
}
