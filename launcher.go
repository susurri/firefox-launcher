package launcher

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/susurri/firefox-launcher/xwindow"
	"gopkg.in/ini.v1"
)

// Mode represents mode of firefox instance.
type Mode int

// Mode definitions for firefox instance.
const (
	Auto Mode = iota
	On
	Off
	Suspend
	None
)

func (m Mode) String() string {
	switch m {
	case Auto:
		return "Auto"
	case On:
		return "On"
	case Off:
		return "Off"
	case Suspend:
		return "Suspend"
	case None:
		return "None"
	default:
		return ""
	}
}

func strToMode(s string) (Mode, error) {
	switch s {
	case "Auto":
		return Auto, nil
	case "On":
		return On, nil
	case "Off":
		return Off, nil
	case "Suspend":
		return Suspend, nil
	case "None":
		return None, nil
	default:
		return None, fmt.Errorf("Invalid mode string %s found", s)
	}
}

// Config configures mode of firefox instance.
type Config struct {
	Name string
	Mode string
}

// Configs is a slice of Config.
type Configs []Config

// Len returns the length of Configs.
func (c Configs) Len() int {
	return len(c)
}

// Less returns if the first element is less than the second one.
func (c Configs) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

// Swap swaps the elements.
func (c Configs) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

const programName = "firefox-launcher"

// Profile of firefox.
type Profile struct {
	Name       string
	IsRelative int
	Path       string
}

// Status is status of firefox process.
type Status int

// Firefox process is runnnig or not.
const (
	Up Status = iota
	Down
)

func (s Status) String() string {
	switch s {
	case Up:
		return "Up"
	case Down:
		return "Down"
	default:
		return ""
	}
}

var (
	firefoxHome    string
	channelCommand chan string
	channelAck     chan string
)

func getFirefoxHome() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(homedir, ".mozilla", "firefox")
}

func loadProfiles() map[string]Profile {
	profiles := make(map[string]Profile)
	filename := filepath.Join(firefoxHome, "profiles.ini")

	sections, err := ini.Load(filename)
	if err != nil {
		log.Fatal(err)
	}

	for _, section := range sections.Sections() {
		if strings.HasPrefix(section.Name(), "Profile") {
			p := new(Profile)

			err := section.MapTo(p)
			if err != nil {
				log.Fatal(err)
			}

			profiles[p.Name] = *p
		}
	}

	return profiles
}

func getRealPath(rel int, path string) string {
	var realpath string
	if rel == 1 {
		realpath = filepath.Join(firefoxHome, path)
	} else {
		realpath = path
	}

	return filepath.Join(realpath, "lock")
}

func getPid(path string) (int, error) {
	if _, err := os.Lstat(path); err != nil {
		return -2, fmt.Errorf("getPid: %w", err)
	}

	link, err := os.Readlink(path)
	if err != nil {
		log.Fatal(err)
	}

	pid, err := strconv.Atoi(strings.Split(link, "+")[1])
	if err != nil {
		return -2, fmt.Errorf("getPid: %w", err)
	}

	return pid, nil
}

func getStatus(pid int) Status {
	exist, err := process.PidExists(int32(pid))
	if err != nil {
		log.Fatal(err)
	}

	if exist {
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			log.Fatal(err)
		}

		if exe, err := p.Exe(); err == nil {
			if exe == "/usr/lib/firefox/firefox" || exe == "/usr/lib64/firefox/firefox" {
				return Up
			}
		}
	}

	return Down
}

func createFirefoxMap(ps *map[string]Profile, cm *map[string]Mode) FirefoxMap {
	firefox := NewFirefoxMap()

	for _, p := range *ps {
		mode, ok := (*cm)[p.Name]
		if !ok {
			mode = None
		}

		var f Firefox

		f.Mode = mode
		f.RealPath = getRealPath(p.IsRelative, p.Path)
		f.Update()
		firefox[p.Name] = f
	}

	return firefox
}

func commandExecutor(input *string, ff *FirefoxMap) {
	words := strings.Fields(*input)
	command := words[0]
	configs := make(Configs, len(*ff))
	i := 0

	for k := range *ff {
		configs[i] = Config{Name: k, Mode: fmt.Sprint((*ff)[k].Mode)}
		i++
	}

	sort.Sort(configs)

	switch command {
	case "exit":
		os.Exit(0)
	case "list":
		const padding = 3
		w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)

		for _, c := range configs {
			fmt.Fprintf(w, "%s\t%s\t(%s)\n", c.Name, c.Mode, (*ff)[c.Name].Status)
		}

		w.Flush()
	case "save":
		saveConfig(configs)
	case "set":
		p := words[1]

		_, ok := (*ff)[p]
		if !ok {
			fmt.Printf("No profile %s found\n", p)
			return
		}

		mode, err := strToMode(strings.Title(words[2]))
		if err == nil {
			(*ff)[p] = Firefox{Pid: (*ff)[p].Pid, Status: (*ff)[p].Status, RealPath: (*ff)[p].RealPath, Mode: mode}
		}
	case "shutdown":
		for k := range *ff {
			(*ff)[k] = Firefox{Pid: (*ff)[k].Pid, Status: (*ff)[k].Status, RealPath: (*ff)[k].RealPath, Mode: Off}
		}
	default:
		return
	}
}

func startFirefox(name string) {
	cmd := exec.Command("setsid", "-f", "firefox", "--no-remote", "-P", name)
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
}

func applyFirefoxMode(ff *FirefoxMap) {
	for k, v := range *ff {
		v.Update()

		switch v.Mode {
		case Auto:
			if v.Status == Down {
				startFirefox(k)
			}

			if v.IsFront() {
				err := syscall.Kill(-v.Pid, syscall.SIGCONT)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				v.Suspend()
			}
		case On:
			switch v.Status {
			case Up:
				_ = syscall.Kill(-v.Pid, syscall.SIGCONT)
			case Down:
				startFirefox(k)
			}
		case Off:
			xwindow.UpdatePidWindowMap()
			v.Shutdown()
		case Suspend:
			if v.Status == Up {
				v.Suspend()
			}
		case None:
		}

		v.Update()
	}
}

func launcherLoop(ps *map[string]Profile, cm *map[string]Mode) {
	firefox := createFirefoxMap(ps, cm)
	commandReceived := false

	prevActiveWindowID, err := xwindow.ActiveWindowID()
	if err != nil {
		log.Fatal("failed to get ActiveWindowID\n")
	}

	for {
		time.Sleep(time.Second)

		a, err := xwindow.ActiveWindowID()
		if err != nil {
			continue
		}

		if commandReceived || prevActiveWindowID != a {
			applyFirefoxMode(&firefox)
		}

		prevActiveWindowID = a
		select {
		case input := <-channelCommand:
			commandExecutor(&input, &firefox)
			channelAck <- "done"

			commandReceived = true
		default:
			commandReceived = false
		}
	}
}

func initVars() {
	firefoxHome = getFirefoxHome()
}

// Run is the entry point.
func Run() {
	initVars()
	writepid()

	profilemap := loadProfiles()
	configs := loadConfig()
	configmap := configsToMap(&configs)
	channelCommand = make(chan string)
	channelAck = make(chan string)

	xwindow.Init()

	go launcherLoop(&profilemap, &configmap)
	promptLoop()
}
