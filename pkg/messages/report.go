package messages

import printerpb "github.com/eolymp/go-sdk/eolymp/printer"

func Report(status printerpb.Job_Status) *printerpb.PrinterConnectorClientMessage {
	return &printerpb.PrinterConnectorClientMessage{
		Message: &printerpb.PrinterConnectorClientMessage_Report_{
			Report: &printerpb.PrinterConnectorClientMessage_Report{
				Status: status,
			},
		},
	}
}
