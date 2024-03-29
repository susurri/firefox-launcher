package xwindow

/*
#cgo LDFLAGS: -lX11
#include <stdio.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>

#define MAXBUFLEN 1024

int cerrorHandler(Display* disp, XErrorEvent* xev) {
	char buf[MAXBUFLEN];
	XGetErrorText(disp, xev -> error_code, buf, MAXBUFLEN);
	fprintf(stderr, "Error in X11 : %s\n", buf);
	return 0;
}

void set_error_handler() {
	XSetErrorHandler(cerrorHandler);
}
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"
)

var (
	display      *C.Display
	root         C.Window
	pidWindowMap map[int]uint
)

func Init() {
	display = C.XOpenDisplay(nil)
	root = C.XDefaultRootWindow(display)

	if display == (*C.Display)(C.NULL) {
		log.Fatal("cannot open Display\n")
	}

	_ = C.set_error_handler()

	pidWindowMap = make(map[int]uint)
}

func getWindowProperty(window C.Window, propName string) (*C.uchar, uint, error) {
	var (
		actualTypeReturn               C.Atom
		actualFormatReturn             C.int
		nitemsReturn, bytesAfterReturn C.ulong
		propReturn                     *C.uchar
	)

	property := C.XInternAtom(display, C.CString(propName), C.True)
	cerr := C.XGetWindowProperty(display, window, property, 0, 1024, C.False, C.AnyPropertyType, &actualTypeReturn, &actualFormatReturn, &nitemsReturn, &bytesAfterReturn, &propReturn)

	if cerr != C.Success {
		return (*C.uchar)(C.NULL), 0, fmt.Errorf("Failed to get property %s", propName)
	}

	return propReturn, uint(nitemsReturn), nil
}

func ActiveWindowID() (uint, error) {
	prop, _, err := getWindowProperty(root, "_NET_ACTIVE_WINDOW")
	if err != nil {
		return 0, fmt.Errorf("Failed to get active window")
	}

	id := *(*C.Window)(unsafe.Pointer(prop))

	C.XFree(unsafe.Pointer(prop))

	return uint(id), nil
}

func getWindowPid(w uint) (int, error) {
	prop, _, err := getWindowProperty((C.Window)(w), "_NET_WM_PID")
	if err != nil {
		return 0, fmt.Errorf("No pid was found for window ID %d", w)
	}

	pid := *(*C.ulong)(unsafe.Pointer(prop))

	C.XFree(unsafe.Pointer(prop))

	return int(pid), nil
}

func PidOfFrontWindow() (int, error) {
	w, err := ActiveWindowID()
	if err != nil {
		return 0, fmt.Errorf("No Active Window found")
	}

	return getWindowPid(w)
}

func PidToWindowID(pid int) (uint, error) {
	w, ok := pidWindowMap[pid]
	if ok {
		return w, nil
	} else {
		return 0, fmt.Errorf("No window found for pid %d", pid)
	}
}

func getWindowList() ([]uint, error) {
	prop, nitems, err := getWindowProperty(root, "_NET_CLIENT_LIST")
	if err != nil {
		return []uint{}, fmt.Errorf("Failed to get window list")
	}

	windows := make([]C.Window, nitems)
	propsize := (C.ulong)(unsafe.Sizeof(windows[0])) * (C.ulong)(nitems)

	C.memcpy(unsafe.Pointer(&windows[0]), unsafe.Pointer(prop), propsize)
	C.XFree(unsafe.Pointer(prop))

	windowlist := make([]uint, nitems)

	for i := range windows {
		windowlist[i] = uint(windows[i])
	}

	return windowlist, nil
}

func UpdatePidWindowMap() {
	pidWindowMap = make(map[int]uint)

	if wl, err := getWindowList(); err == nil {
		for _, w := range wl {
			if pid, err := getWindowPid(w); err == nil {
				pidWindowMap[pid] = w
			}
		}
	}
}

func CloseWindowByPid(pid int) error {
	var xev C.XEvent
	xcmev := (*C.XClientMessageEvent)((unsafe.Pointer)(&xev))
	atom := C.XInternAtom(display, C.CString("_NET_CLOSE_WINDOW"), C.True)
	window := (C.Window)(pidWindowMap[pid])
	xcmev._type = C.ClientMessage
	xcmev.serial = 0
	xcmev.send_event = C.True
	xcmev.display = display
	xcmev.window = window
	xcmev.message_type = atom
	xcmev.format = 32

	var data [(unsafe.Sizeof(xcmev.data))]byte = xcmev.data
	for i := range data {
		data[i] = 0
	}

	switch C.XSendEvent(display, root, C.False,
		(C.SubstructureNotifyMask | C.SubstructureRedirectMask),
		&xev) {
	case 0:
		return fmt.Errorf("XSendEvent conversion failed")
	case C.BadValue:
		return fmt.Errorf("XSendEvent returned BadValue")
	case C.BadWindow:
		return fmt.Errorf("XSendEvent returned BadWindow")
	default:
		return nil
	}
}
