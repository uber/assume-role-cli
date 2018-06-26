package assumerole

import (
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
)

// Config is the config for the AssumeRole app.
type Config struct {
	// RefreshBeforeExpiry is a duration prior to the credentials expiring
	// where we'll refresh them anyway. This is to prevent a command running
	// just before credentials are about to expire. Defaults to 15m.
	RefreshBeforeExpiry time.Duration `json:"refresh_before_expiry"`

	// RolePrefix allows the user to specify a prefix for the role ARN that
	// will be combined with what is specified as the role when executing the
	// app. For example, if the prefix is "arn:aws:iam::123:role/" and the user
	// executes the app with role "foobar", the final ARN will become:
	// "arn:aws:iam::123:role/foobar".
	RolePrefix string `json:"role_prefix"`

	// ProfileNamePrefix is a prefix that will prepended to the role name to
	// create the profile name under which the AWS configuration will be saved.
	ProfileNamePrefix string `json:"profile_name_prefix"`
}

// SetDefaults sets any default values for unset variables.
func (c *Config) setDefaults() {
	if c.RefreshBeforeExpiry == 0 {
		c.RefreshBeforeExpiry = time.Minute * 15
	}
}

// LoadConfig reads config values from a file and returns the config.
func LoadConfig(configFilePath string) (*Config, error) {
	var config Config

	b, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
