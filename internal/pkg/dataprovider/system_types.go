package dataprovider

// Inherits DataProvider interface
type SystemInfoProvider struct {
}

type CPUGuestStatDetails struct {
	CpuNo   int64
	Name    string
	CpuMode string
	Value   float64
}

type CPUStatDetails struct {
	CpuNo   int64
	Name    string
	CpuMode string
	Value   float64
}
