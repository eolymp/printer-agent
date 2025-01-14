package ipp

import "github.com/OpenPrinting/goipp"

type PrinterAttributes struct {
	Name                  string
	Info                  string
	State                 PrinterState
	StateReason           string
	QueuedJobCount        int
	ColorSupported        bool
	OperationsSupported   []goipp.Op
	PageDeliverySupported []string
}
