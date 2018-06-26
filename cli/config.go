package cli

import (
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func findConfigFile() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for _, path := range searchPaths(wd) {
		configFile := filepath.Join(path, "assume-role.yaml")
		if fileExists(configFile) {
			return configFile, nil
		}
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	configFile := filepath.Join(home, ".aws", "assume-role.yaml")
	if fileExists(configFile) {
		return configFile, nil
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
