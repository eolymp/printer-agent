package messages

import printerpb "github.com/eolymp/go-sdk/eolymp/printer"

func Authenticate(secret string) *printerpb.PrinterConnectorClientMessage {
	return &printerpb.PrinterConnectorClientMessage{
		Message: &printerpb.PrinterConnectorClientMessage_Authenticate_{
			Authenticate: &printerpb.PrinterConnectorClientMessage_Authenticate{
				Secret: secret,
			},
		},
	}
}
