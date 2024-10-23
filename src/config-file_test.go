package main

import "reflect"
import "testing"
import "github.com/stretchr/testify/assert"

func Test_readConfig(t *testing.T) {
	t.Run("non-existent config file", func(t *testing.T) {
		cfg, err := readConfig("/no/such/file.json")
		var nilConfig *config
		assert.Equal(t, nilConfig, cfg)
		assert.Nil(t, cfg)
		assert.Error(t, err, "open /no/such/file.json: no such file or directory")
	})

	t.Run("malformed config file", func(t *testing.T) {
		cfg, err := readConfig("../etc/not-json.txt")
		var nilConfig *config
		assert.Equal(t, nilConfig, cfg)
		assert.Error(t, err, "invalid character 'T' looking for beginning of value")
	})

	t.Run("well-known config file", func(t *testing.T) {
		cfg, err := readConfig("../etc/silent.json")
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(cfg, &config{
			Logging: loggingConfig{},
			Listen: listenConfig{
				Host: "0.0.0.0",
				Port: 12369,
			},
		}))
	})
}
