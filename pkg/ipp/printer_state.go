package ipp

type PrinterState int

func (s PrinterState) String() string {
	switch s {
	case PrinterIdle:
		return "idle"
	case PrinterProcessing:
		return "processing"
	case PrinterStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

const (
	PrinterIdle       PrinterState = 3
	PrinterProcessing PrinterState = 4
	PrinterStopped    PrinterState = 5
)
