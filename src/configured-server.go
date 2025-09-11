package main

import "fmt"
import "regexp"
import "github.com/MikeTaylor/catlogger"

func MakeConfiguredServer(configFile string, httpRoot string) (*ModReportingServer, error) {
	var cfg *config
	cfg, err := readConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file '%s': %w", configFile, err)
	}

	cl := cfg.Logging
	logger := catlogger.MakeLogger(cl.Categories, cl.Prefix, cl.Timestamp)
	logger.AddTransformation(regexp.MustCompile(`\\"pass\\":\\"[^"]*\\"`), `\"pass\":\"********\"`);
	logger.Log("config", fmt.Sprintf("%+v", cfg))

	server := MakeModReportingServer(cfg, logger, httpRoot)
	return server, nil
}
