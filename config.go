/*
 *  Copyright (c) 2018 Uber Technologies, Inc.
 *
 *     Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
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
