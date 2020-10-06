package launcher

import (
	"os/exec"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
)

// Firefox holds the status of the firefox process
type Firefox struct {
  Pid int
  Status Status
  RealPath string
  Mode Mode
}

// Update updates the Firefox
func (f *Firefox) Update() {
	pid, err := getPid(f.RealPath)
	if err != nil {
	  f.Status = Down
	} else {
    f.Pid = pid
	  f.Status = getStatus(pid)
	}
	return
}

// IsFront returns whether it is running at the front window
func (f Firefox) IsFront() bool {
	p, err := pidOfFrontWindow()
	if err != nil {
		return false
	}
	pg, err := syscall.Getpgid(p)
	if err != nil {
		return false
	}
	return pg == f.Pid
}

// Suspend suspends the firefox
func (f Firefox) Suspend() {
	p, err := process.NewProcess(int32(f.Pid))
	if err != nil { return }
	s, err := p.Status()
	if err != nil || s == "T" {
		return
	}
	ctime, err := p.CreateTime()
	if err != nil { return }
	if time.Now().Unix() * 1000 - ctime > 300 * 1000 {
		syscall.Kill(-f.Pid, syscall.SIGSTOP)
	}
}

// Shutdown gracefully shutdowns the firefox
func (f Firefox) Shutdown() {
	w, err := pidToWindowID(f.Pid)
	if err == nil {
		syscall.Kill(-f.Pid, syscall.SIGCONT)
		exec.Command("wmctrl", "-i", "-c", w).Run()
	}
}

// FirefoxMap is a map from profile name to Firefox
type FirefoxMap map[string]Firefox

// NewFirefoxMap creates a FirefoxMap
func NewFirefoxMap() FirefoxMap {
  return make(FirefoxMap)
}
