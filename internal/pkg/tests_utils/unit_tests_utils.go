package tests_utils

import (
	"fmt"
	"os"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
)

/*
Constants, common variables and helper functions
*/

const (
	TESTS_DEFAULT_CONFIG_FILE     = "tests_data/default_ape.toml"
	TESTS_USERS_CONFIG_FILE       = "tests_data/users_ape.toml"
	TESTS_MOCK_CONFIG_FILE        = "tests_data/mock_ape.toml"
	TESTS_DEFAULT_GAUGE_LIST_FILE = "configs/gauge_stats_list.toml"
)

func GetConfigfileLocation(filename string) string {
	l_filename, _ := os.Getwd()

	// l_filename = l_filename + "/../../../../" + filename
	filePath := l_filename + "/" + filename

	if _, err := os.Stat(filePath); err != nil {
		filePath = l_filename + "/../../../../" + filePath
		fmt.Println(filePath, " - Given path not found: ", filename, " \n\tHence using ", filePath)
	}

	return filePath
}

func InitConfigurations(config_filename string) {
	config.InitConfig(GetConfigfileLocation(config_filename))
	config.InitGaugeStats(GetConfigfileLocation(TESTS_DEFAULT_GAUGE_LIST_FILE))

}
