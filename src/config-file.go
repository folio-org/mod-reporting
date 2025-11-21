package main

import "os"
import "io"
import "encoding/json"
import "strconv"

type loggingConfig struct {
	Categories string `json:"categories"`
	Prefix     string `json:"prefix"`
	Timestamp  bool   `json:"timestamp"`
}

type listenConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type reportUrlWhitelistConfig []string

type config struct {
	Logging            loggingConfig            `json:"logging"`
	Listen             listenConfig             `json:"listen"`
	QueryTimeout       int                      `json:"queryTimeout"`
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

	queryTimeoutString := os.Getenv("MOD_REPORTING_QUERY_TIMEOUT")
	if queryTimeoutString != "" {
		cfg.QueryTimeout, _ = strconv.Atoi(queryTimeoutString)
	} else if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = 60
	}

	return &cfg, nil
}
