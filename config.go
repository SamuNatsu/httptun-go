package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode       string
	Port       uint16
	RemoteAddr string `yaml:"remote-addr"`
}

func ParseConfig() (Config, error) {
	cfgIn, err := os.Open("config.yaml")
	if err != nil {
		return Config{}, err
	}

	ret := Config{}
	decoder := yaml.NewDecoder(cfgIn)
	err = decoder.Decode(&ret)
	if err != nil {
		return Config{}, err
	}

	modeSet := map[string]struct{}{
		"server": {},
		"client": {},
	}
	if _, ok := modeSet[ret.Mode]; !ok {
		return Config{}, fmt.Errorf("invalid mode: %s", ret.Mode)
	}

	if ret.Port < 1 {
		return Config{}, fmt.Errorf("invalid port: %d", ret.Port)
	}

	return ret, nil
}
