package launcher

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/c-bata/go-prompt"
)

// Help holds a help message
type Help struct {
	Text string
	Description string
}

func validateCommand(input string) bool {
	words := strings.Fields(input)
	if len(words) < 1 { return false}
	command := words[0]
	switch command {
	case "help":
		help()
		return true
	case "exit", "list", "shutdown", "save", "start":
		if len(words) > 1 {
			return false
		}
		return true
	case "set":
		ok := false
		modes :=  []string{"auto", "on", "off", "suspend", "none"}
		for _, w := range modes {
      if w == words[2] { ok = true; break }
		}
		if !ok {
			return false
		}
		ok = false
		for _, p := range profileSuggest {
			if p.Text == words[1] { ok = true; break }
		}
		if !ok {
			return false
		}
		return true
	default:
		return false
	}
}

func help() {
	command := []Help{
		{Text: "exit", Description: "Exit from the launcher"},
		{Text: "list", Description: "Show profiles, configs and statuses"},
		{Text: "save", Description: "Save configs"},
		{Text: "set <profile> <mode>", Description: "Set mode"},
		{Text: "shutdown", Description: "Shutdown firefox"},
		{Text: "start", Description: "Start firefox"},
	}
	mode := []Help{
		{Text: "auto", Description: "Auto mode"},
		{Text: "on", Description: "Always on"},
		{Text: "off", Description: "Always off"},
		{Text: "suspend", Description: "Always suspend"},
		{Text: "none", Description: "Leave it as is"},
	}
	fmt.Println("")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, c := range command {
		fmt.Fprintf(w, "%s\t%s\n", c.Text, c.Description)
	}
	w.Flush()
	fmt.Println("")
	fmt.Println("modes")
	fmt.Println("-----------------------")
	for _, c := range mode {
		fmt.Fprintf(w, "%s\t%s\n", c.Text, c.Description)
	}
	w.Flush()
}

func executor(input string) {
	if !validateCommand(input) {
		fmt.Printf("Invalid command : %s\n", input)
		help()
		return
	}
	channelCommand <- input
  ack := <- channelAck
	if ack != "done" {
		fmt.Println("Command not executed")
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	profile := profileSuggest
	command := []prompt.Suggest{
		{Text: "exit", Description: "Exit from the launcher"},
		{Text: "list", Description: "Show profiles and configs"},
		{Text: "save", Description: "Save configs"},
		{Text: "set", Description: "Set mode"},
		{Text: "shutdown", Description: "Shutdown firefox"},
		{Text: "start", Description: "Start firefox"},
	}
	mode := []prompt.Suggest{
		{Text: "auto", Description: "Auto mode"},
		{Text: "on", Description: "Always on"},
		{Text: "off", Description: "Always off"},
		{Text: "suspend", Description: "Always suspend"},
		{Text: "none", Description: "Leave it as is"},
	}
	words := strings.Fields(d.Text)
	wordcount := len(words)
	b := d.GetWordBeforeCursor()
	switch {
	case (wordcount == 0 || (wordcount == 1 && words[0] == b)):
		return prompt.FilterHasPrefix(command, b, true)
	case (words[0] == "set" && (wordcount == 1 || (wordcount == 2 && words[1] == b))):
		return prompt.FilterHasPrefix(profile, b, true)
	case (words[0] == "set" && (wordcount == 2 || (wordcount == 3 && words[2] == b))):
		return prompt.FilterHasPrefix(mode, b, true)
	default:
		return nil
	}
}

func promptLoop(pm bool) {
	if pm  {
		var history []string
		p := prompt.New(
			executor,
			completer,
			prompt.OptionHistory(history),
		)
		p.Run()
		return
  }
	var s string
	for {
		fmt.Print("> ")
		fmt.Scanln(&s)
		executor(s)
	}
}
