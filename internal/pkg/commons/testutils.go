package commons

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
	if _, err := os.Stat(filename); err != nil {
		l_filename, _ := os.Getwd()

		filePath := l_filename + "/" + filename

		if _, err := os.Stat(filePath); err != nil {
			filePath = l_filename + "/../../../../" + filename
			fmt.Println("==> Given filename not found: ", filename, " \n\tHence using ", filePath)
		}

		return filePath
	}

	return filename
}

func InitConfigurations(config_filename string) {
	config.InitConfig(GetConfigfileLocation(config_filename))
	config.InitGaugeStats(GetDefaultGaugeListFilename())

}

func GetDefaultGaugeListFilename() string {
	return GetExporterBaseFolder() + "/" + TESTS_DEFAULT_GAUGE_LIST_FILE
}

func GetWatchersConfigFile(filename string) string {
	return GetExporterBaseFolder() + "/internal/pkg/statprocessors/tests/" + filename
}

func GetWatchersMockResultsFile(filename string) string {
	return GetExporterBaseFolder() + "/internal/pkg/statprocessors/tests/" + filename
}

func GetExecutorsConfigFile(filename string) string {
	return GetExporterBaseFolder() + "/internal/pkg/executors/tests/" + filename
}

func GetPromMockResultsFile(filename string) string {
	return GetExporterBaseFolder() + "/internal/pkg/executors/tests/" + filename
}
