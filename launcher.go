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

	"github.com/c-bata/go-prompt"
	"github.com/shirou/gopsutil/process"
	"gopkg.in/ini.v1"
)

// Mode represents mode of firefox instance
type Mode int

// Mode definitions for firefox instance
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
// Config configures mode of firefox instance
type Config struct {
	Name string
	Mode string
}

// Configs is a slice of Config
type Configs []Config

// Len returns the length of Configs
func (c Configs) Len() int {
	return len(c)
}

// Less returns if the first element is less than the second one
func (c Configs) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

// Swap swaps the elements
func (c Configs) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

const programName = "firefox-launcher"

// Profile of firefox
type Profile struct {
	Name string
	IsRelative int
	Path string
}

// Status is status of firefox process
type Status int

// Firefox proccess is runnnig or not
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
	firefoxHome string
	profileSuggest []prompt.Suggest
	channelCommand chan string
	channelAck chan string
)

func getFirefoxHome() string {
	homedir, err := os.UserHomeDir()
	if err != nil { log.Fatal(err) }
	return filepath.Join(homedir, ".mozilla", "firefox")
}

func loadProfiles() map[string]Profile {
	profiles := make(map[string]Profile)
	filename := filepath.Join(firefoxHome, "profiles.ini")
	sections, err := ini.Load(filename)
	if err != nil { log.Fatal(err) }
	for _, section := range sections.Sections() {
		if strings.HasPrefix(section.Name(), "Profile") {
			p := new(Profile)
			err := section.MapTo(p)
			if err != nil { log.Fatal(err) }
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
		return -2, err
	}
	link, err := os.Readlink(path)
	if err != nil { log.Fatal(err) }
	pid, err := strconv.Atoi(strings.Split(link, "+")[1])
	if err != nil {
		return -2, err
	}
	return pid, err
}

func getStatus(pid int) Status {
	exist, err := process.PidExists(int32(pid))
	if err != nil { log.Fatal(err) }
	if exist {
		p, err := process.NewProcess(int32(pid))
		if err != nil { log.Fatal(err) }
		if exe, err := p.Exe(); err == nil {
			if exe == "/usr/lib/firefox/firefox" {
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
		if !ok { mode = None }
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
		configs[i] = Config{ Name: k, Mode: fmt.Sprint((*ff)[k].Mode) }
		i++
	}
	sort.Sort(configs)
	switch command {
	case "exit":
		os.Exit(0)
	case "list":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		for _, c := range configs {
			fmt.Fprintf(w, "%s\t%s\t(%s)\n", c.Name, c.Mode, (*ff)[c.Name].Status)
		}
		w.Flush()
	case "save":
		saveConfig(configs)
	case "set":
		p := words[1]
		mode, err := strToMode(strings.Title(words[2]))
		if err == nil {
			(*ff)[p] = Firefox{Pid: (*ff)[p].Pid, Status: (*ff)[p].Status, RealPath: (*ff)[p].RealPath, Mode: mode}
		}
	default:
		return
	}
}

func startFirefox(name string) {
	cmd := exec.Command("setsid", "firefox", "--no-remote", "-P", name)
	err := cmd.Start()
	if err != nil { log.Fatal(err) }
}

func applyFirefoxMode(ff *FirefoxMap) {
	for k, v := range *ff {
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
				syscall.Kill(-v.Pid, syscall.SIGCONT)
			case Down:
				startFirefox(k)
			}
		case Off:
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
	prevActiveWindowID := activeWindowID()
	for {
		time.Sleep(1000 * time.Millisecond)
		a := activeWindowID()
		if commandReceived || prevActiveWindowID != a {
			applyFirefoxMode(&firefox)
		}
		prevActiveWindowID = a
		select {
		case input := <- channelCommand:
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

func makeSuggest(pm *map[string]Profile) []prompt.Suggest {
	var s []prompt.Suggest
	for p := range *pm {
		s = append(s, prompt.Suggest{Text: p, Description: "profile " + p })
	}
	return s
}

// Run is the entry point
func Run() {
	initVars()
	fd := lockpid(openpid())
	defer syscall.Close(fd)
	profilemap := loadProfiles()
	configs := loadConfig()
	configmap := configsToMap(&configs)
	profileSuggest = makeSuggest(&profilemap)
	channelCommand = make(chan string)
	channelAck = make(chan string)
	go launcherLoop(&profilemap, &configmap)
	promptLoop(false)
}
