package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-ini/ini"
)

const (
	// configFilename is the config filename for the distributor program.
	configFilename = "distributor.conf"

	// mappingSection is the section name of mapping list.
	mappingSection = "Mapping"
)

// loadConfig load distributor mapping list from configuration file.
func loadConfig() (map[int]string, error) {
	cfg, err := ini.Load(configFilename)
	if err != nil {
		return nil, err
	}

	// Use port=host:port to mapping.
	keys := cfg.Section(mappingSection).Keys()
	if len(keys) == 0 {
		return nil, errors.New("no mapping list configured")
	}

	mapping := make(map[int]string, len(keys))
	for _, key := range keys {
		port, err := strconv.Atoi(key.Name())
		if err != nil {
			return nil, fmt.Errorf("invalid port value %s", key.Name())
		}
		mapping[port] = key.Value()
	}

	return mapping, nil
}
