package unittests

import (
	"os"
)

/*
Constants, common variables and helper functions
*/

const (
	DEFAULT_CONFIG_FILE = "tests/default_ape.toml"
	MOCK_CONFIG_FILE    = "tests/mock_ape.toml"
)

func GetConfigfileLocation(filename string) string {
	l_filename, _ := os.Getwd()

	l_filename = l_filename + "/../../../" + filename

	return l_filename
}
