package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Server contains all configuration settings related to server binary.
type Server struct {
	API Api `yaml:"api"`
}

// Guide section describe settings for guide host.
type Guide struct {
	// Host is an address of guide instance.
	Host string
	// Key is a secret key shared between server and guide.
	Key string
}

// Api section describes settings for API.
type Api struct {
	// HTTPAddr is TCP address service's API listens on.
	HTTPAddr string `yaml:"http_addr"`
}

// Validate validates some configuration settings to catch configuration errors early.
func (cfg *Server) Validate() error {
	return nil
}

// Parse YAML configuration file.
func Parse(filePath string) (Server, error) {
	c := Server{}

	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return c, fmt.Errorf("cannot read from file: %s, err: %w", filePath, err)
	}

	d := yaml.NewDecoder(f)
	// Because there are multiple consumers of the configuration being parsed here, it is much simpler to ignore extra fields
	// compared to adjusting configuration source (Helm, inventory folder) to contain only those fields we've defined in this code.
	d.SetStrict(false)

	err = d.Decode(&c)
	if err != nil {
		return c, fmt.Errorf("error decoding file: %s, err: %w", filePath, err)
	}

	err = f.Close()
	if err != nil {
		return c, fmt.Errorf("cannot close file: %s, err: %w", filePath, err)
	}

	return c, nil
}
