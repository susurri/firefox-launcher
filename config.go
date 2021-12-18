package launcher

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kyoh86/xdg"
)

func createConfig(configDir string, configName string) {
	filename := filepath.Join(configDir, configName)
	if _, err := os.Stat(filename); err == nil {
		return
	}

	if _, err := os.Stat(configDir); err != nil {
		err := os.MkdirAll(configDir, 0o700)
		if err != nil {
			log.Fatal(err)
		}
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Fatal(err)
	}

	buf := []byte("[]")

	_, err = f.Write(buf)
	if err != nil {
		log.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

const configName = "config.json"

func loadConfig() []Config {
	var configs []Config

	configDir := filepath.Join(xdg.ConfigHome(), programName)
	createConfig(configDir, configName)
	filename := filepath.Join(configDir, configName)

	fileinfo, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, fileinfo.Size())

	_, err = f.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(buf, &configs); err != nil {
		log.Fatal(err)
	}

	return configs
}

func configsToMap(configs *[]Config) map[string]Mode {
	configmap := make(map[string]Mode)

	for _, c := range *configs {
		mode, err := strToMode(c.Mode)
		if err != nil {
			log.Fatal(err)
		}

		configmap[c.Name] = mode
	}

	return configmap
}

func configsFromMap(configmap *map[string]Mode) []Config {
	configs := make([]Config, 0, len(*configmap))
	for n, m := range *configmap {
		configs = append(configs, Config{Name: n, Mode: fmt.Sprint(m)})
	}

	return configs
}

func saveConfig(configs Configs) {
	configDir := filepath.Join(xdg.ConfigHome(), programName)
	createConfig(configDir, configName)
	filename := filepath.Join(configDir, configName)
	backup := filename + ".bak"

	if err := os.Rename(filename, backup); err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Fatal(err)
	}

	buf, err := json.MarshalIndent(configs, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.Write(buf)
	if err != nil {
		log.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}
