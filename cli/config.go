package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
)

type accountInfo struct {
	Email string `toml:"email"`
	Token string `toml:"token"`
}

type Config struct {
	Account accountInfo `toml:"account"`
}

const defaultConfigDir = "~/.config/lab47"

func LoadConfig() (*Config, error) {
	var cfg Config

	cfgDir := os.Getenv("LAB47_HOME")
	if cfgDir == "" {
		cfgDir = defaultConfigDir
	}

	path, err := homedir.Expand(filepath.Join(cfgDir, "svc.toml"))
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}

		return nil, err
	}

	_, err = toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	cfgDir := os.Getenv("LAB47_HOME")
	if cfgDir == "" {
		cfgDir = defaultConfigDir
	}

	path, err := homedir.Expand(filepath.Join(cfgDir, "svc.toml"))
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}

	w, err := os.Create(path)
	if err != nil {
		return err
	}

	defer w.Close()

	return toml.NewEncoder(w).Encode(cfg)
}
