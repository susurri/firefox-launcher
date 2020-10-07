package launcher

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func activeWindowID() string {
  out, err := exec.Command("/usr/bin/xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
  if err != nil {
    return ""
  }
	w := strings.Fields(string(out))
  return w[len(w) - 1]
}

func windowIDToPid(windowID string) (int, error){
	out, err := exec.Command("/usr/bin/xprop", "-id", windowID, "_NET_WM_PID").Output()
  if err != nil {
		return -1, fmt.Errorf("No Window with ID %s found", windowID)
	}
	w := strings.Fields(string(out))
  return strconv.Atoi(w[len(w) - 1])
}

func pidOfFrontWindow() (int, error) {
	return windowIDToPid(activeWindowID())
}

func pidToWindowID(pid int) (string, error) {
	out, err := exec.Command("wmctrl", "-lp").Output()
  if err == nil {
		lines := strings.Split(string(out),"\n")
		for _, line := range(lines) {
			words := strings.Fields(line)
			if len(words) < 3 { break }
			if p, err := strconv.Atoi(words[2]); err == nil && p == pid {
				return words[0], nil
			}
		}
	}
	return "", fmt.Errorf("No Window with pid %d found", pid)
}
