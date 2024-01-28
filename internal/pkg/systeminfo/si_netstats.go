package systeminfo

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func GetNetStatnfo() []SystemInfoStat {
	arrSysInfoStats := parseNetStats(GetProcFilePath("net/netstat"))
	return arrSysInfoStats
}

func parseNetStats(fileName string) []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		nameParts := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		valueParts := strings.Split(scanner.Text(), " ")
		protocol := nameParts[0][:len(nameParts[0])-1]
		if len(nameParts) != len(valueParts) {
			return arrSysInfoStats
		}
		for i := 1; i < len(nameParts); i++ {
			if acceptNetstat(nameParts[i]) {
				fmt.Println("protocol: ", protocol, " name: ", nameParts[i], " value: ", valueParts[i])
			} else {
				fmt.Println("IGNORED name: ", nameParts[i], " value: ", valueParts[i])
			}
		}
	}

	return arrSysInfoStats
}
