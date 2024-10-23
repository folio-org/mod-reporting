package main

import "testing"
import "github.com/stretchr/testify/assert"

func Test_MakeConfiguredServer(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		server, err := MakeConfiguredServer("../etc/silent.json", ".")
		assert.Nil(t, err)
		assert.NotNil(t, server)
	})

	t.Run("unsuccessful", func(t *testing.T) {
		server, err := MakeConfiguredServer("/no/such/file.json", ".")
		assert.Nil(t, server)
		assert.Regexp(t, "cannot read config file", err.Error())
	})
}
