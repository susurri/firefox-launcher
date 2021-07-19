package xwindow

import (
	"fmt"
	"os"
	"strconv"

	x "github.com/linuxdeepin/go-x11-client"
	"github.com/linuxdeepin/go-x11-client/util/wm/ewmh"
)

var (
	conn *x.Conn
)

func Init() {
	var err error
	conn, err = x.NewConn()
	if err != nil {
		_ = os.Exit
	}
}

func ActiveWindowID() string {
	out, err := ewmh.GetActiveWindow(conn).Reply(conn)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(uint64(out), 16)
}

func PidOfFrontWindow() (uint32, error) {
	w := ActiveWindowID()
	if w == "" {
		return 0, fmt.Errorf("No Active Window found")
	}
	windowID, err := strconv.ParseUint(w, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("No Window with ID %s found", w)
	}
	return ewmh.GetWMPid(conn, x.Window(windowID)).Reply(conn)
}

func PidToWindowID(pid int) (string, error) {
	cookie := ewmh.GetClientList(conn)
	windows, err := cookie.Reply(conn)
	if err != nil {
		_ = os.Exit
	}
	for _, w := range windows {
		if p, err := ewmh.GetWMPid(conn, w).Reply(conn); err == nil && int(p) == pid {
			return strconv.FormatUint(uint64(w), 16), nil
		}
	}
	return "", fmt.Errorf("No Window with pid %d found", pid)
}
