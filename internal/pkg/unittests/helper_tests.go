package unittests

import (
	"os"
)

/*
Constants, common variables and helper functions
*/

const (
	DEFAULT_CONFIG_FILE     = "tests/default_ape.toml"
	DEFAULT_GAUGE_LIST_FILE = "gauge_stats_list.toml"
)

func GetConfigfileLocation(filename string) string {
	l_filename, _ := os.Getwd()

	l_filename = l_filename + "/../../../" + filename

	return l_filename
}
