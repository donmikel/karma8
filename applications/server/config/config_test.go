package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	want := Server{
		API: Api{HTTPAddr: "0.0.0.0:8002"},
	}

	got, err := Parse("config.yml")

	assert.NoError(t, got.Validate())
	assert.Equal(t, nil, err)
	assert.Equal(t, want, got)
}
