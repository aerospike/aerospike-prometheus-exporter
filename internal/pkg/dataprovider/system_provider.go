package dataprovider

import (
	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

func (sip SystemInfoProvider) GetCPUDetails() ([]map[string]float64, []map[string]float64) {

	arrGuestCpuStats := []map[string]float64{}
	arrCpuStats := []map[string]float64{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("parseCpuStats Error while reading CPU Stats from ", PROC_PATH, " Error ", err)
		return arrGuestCpuStats, arrCpuStats
	}

	stats, err := fs.Stat()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return arrGuestCpuStats, arrCpuStats
	}
	// guest_seconds_total:user:100;nice:10
	// seconds_total:

	for index, cpu := range stats.CPU {
		// fmt.Println("parsing CPU stats ", index)
		guestCpuValues := make(map[string]float64)
		guestCpuValues["index"] = float64(index)
		guestCpuValues["user"] = cpu.Guest
		guestCpuValues["nice"] = cpu.GuestNice

		cpuValues := make(map[string]float64)
		cpuValues["index"] = float64(index)
		cpuValues["user"] = cpu.Guest
		cpuValues["idle"] = cpu.Idle
		cpuValues["irq"] = cpu.IRQ
		cpuValues["iowait"] = cpu.Iowait
		cpuValues["nice"] = cpu.Nice
		cpuValues["soft_irq"] = cpu.SoftIRQ
		cpuValues["steal"] = cpu.Steal
		cpuValues["system"] = cpu.System
		cpuValues["user"] = cpu.User

		arrGuestCpuStats = append(arrGuestCpuStats, guestCpuValues)
		arrCpuStats = append(arrCpuStats, cpuValues)

		// arrGuestCpuStatDetails = append(arrGuestCpuStatDetails, CPUGuestStatDetails{index, "guest_seconds_total", "user", cpu.Guest})
		// arrGuestCpuStatDetails = append(arrGuestCpuStatDetails, CPUGuestStatDetails{index, "guest_seconds_total", "nice", cpu.GuestNice})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "idle", cpu.Idle})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "irq", cpu.IRQ})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "iowait", cpu.Iowait})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "nice", cpu.Nice})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "soft_irq", cpu.SoftIRQ})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "steal", cpu.Steal})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "system", cpu.System})
		// arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "user", cpu.User})
	}

	return arrGuestCpuStats, arrCpuStats
}
