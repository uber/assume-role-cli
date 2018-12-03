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
package cli

import (
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
)

const configFile = "assume-role.yaml"
const userPath = ".aws"
const systemPath = "/etc"

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// findConfigFile returns the path of the config file to be used, with precedence:
// 1. Local to project
// 2. Local to user (in ~/.aws DIR)
// 3. System wide
func findConfigFile() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for _, path := range searchPaths(wd) {
		configFile := filepath.Join(path, configFile)
		if fileExists(configFile) {
			return configFile, nil
		}
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	userConfigFile := filepath.Join(home, userPath, configFile)
	if fileExists(userConfigFile) {
		return configFile, nil
	}

	systemConfigFile := filepath.Join(systemPath, configFile)
	if fileExists(systemConfigFile) {
		return systemConfigFile, nil
	}

	return "", nil
}

// searchPaths returns a list of paths from basePath upwards to the root ("/).
func searchPaths(basePath string) (paths []string) {
	root := basePath

	for root != "/" {
		paths = append(paths, root)
		root = filepath.Dir(root)
	}

	paths = append(paths, "/")

	return paths

}
