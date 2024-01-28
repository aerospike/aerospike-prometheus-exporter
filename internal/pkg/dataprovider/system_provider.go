package dataprovider

import (
	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

func (sip SystemInfoProvider) GetCPUDetails() ([]CPUStatDetails, []CPUGuestStatDetails) {

	arrCpuStatDetails := []CPUStatDetails{}
	arrGuestCpuStatDetails := []CPUGuestStatDetails{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("parseCpuStats Error while reading CPU Stats from ", PROC_PATH, " Error ", err)
	}

	stats, err := fs.Stat()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
	}

	for index, cpu := range stats.CPU {
		// fmt.Println("parsing CPU stats ", index)
		arrGuestCpuStatDetails = append(arrGuestCpuStatDetails, CPUGuestStatDetails{index, "guest_seconds_total", "user", cpu.Guest})
		arrGuestCpuStatDetails = append(arrGuestCpuStatDetails, CPUGuestStatDetails{index, "guest_seconds_total", "nice", cpu.GuestNice})

		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "idle", cpu.Idle})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "irq", cpu.IRQ})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "iowait", cpu.Iowait})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "nice", cpu.Nice})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "soft_irq", cpu.SoftIRQ})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "steal", cpu.Steal})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "system", cpu.System})
		arrCpuStatDetails = append(arrCpuStatDetails, CPUStatDetails{index, "seconds_total", "user", cpu.User})
	}

	return arrCpuStatDetails, arrGuestCpuStatDetails
}
