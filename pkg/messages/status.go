package messages

import printerpb "github.com/eolymp/go-sdk/eolymp/printer"

func Status(status printerpb.Printer_Status) *printerpb.PrinterConnectorClientMessage {
	return &printerpb.PrinterConnectorClientMessage{
		Message: &printerpb.PrinterConnectorClientMessage_Status_{
			Status: &printerpb.PrinterConnectorClientMessage_Status{
				Status: status,
			},
		},
	}
}
