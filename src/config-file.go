package main

import "os"
import "io"
import "encoding/json"

type loggingConfig struct {
	Categories string `json:"categories"`
	Prefix     string `json:"prefix"`
	Timestamp  bool   `json:"timestamp"`
}

type listenConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type reportUrlWhitelistConfig [] string

type config struct {
	Logging loggingConfig `json:"logging"`
	Listen  listenConfig  `json:"listen"`
	ReportUrlWhitelist reportUrlWhitelistConfig `json:"reportUrlWhitelist"`
}

func readConfig(name string) (*config, error) {
	jsonFile, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var cfg config
	err = json.Unmarshal(byteValue, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
