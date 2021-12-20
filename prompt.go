package launcher

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Help holds a help message.
type Help struct {
	Text        string
	Description string
}

func validateCommand(input string) bool {
	words := strings.Fields(input)
	if len(words) < 1 {
		return false
	}

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
		if len(words) != 3 {
			return false
		}

		ok := false

		modes := []string{"auto", "on", "off", "suspend", "none"}
		for _, w := range modes {
			if w == words[2] {
				ok = true

				break
			}
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

	const padding = 3
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)

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

	if ack := <-channelAck; ack != "done" {
		fmt.Println("Command not executed")
	}
}

func promptLoop() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")

	for scanner.Scan() {
		s := scanner.Text()
		fmt.Println(s)
		executor(s)
		fmt.Print("> ")
	}
}
