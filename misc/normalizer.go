package misc

import (
	"os/user"
	"strings"
)

// PathNormalize normalizes the path
func PathNormalize(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err != nil {
			return path, err
		}

		path = usr.HomeDir + "/" + strings.TrimPrefix(path, "~/")
	}

	return path, nil
}

// DirNormalize normalizes the directory path
func DirNormalize(path string) string {
	for strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	path += "/"

	return path
}
