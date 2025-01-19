package connector

import (
	"fmt"
	"net/url"
	"strconv"

	printerpb "github.com/eolymp/go-sdk/eolymp/printer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Connect(uri string) (printerpb.PrinterConnectorClient, error) {
	link, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL %q: %v", uri, err)
	}

	port, _ := strconv.Atoi(link.Port())
	if port == 0 {
		if link.Scheme == "http" {
			port = 80
		} else {
			port = 443
		}
	}

	tls := credentials.NewClientTLSFromCert(nil, link.Hostname())

	conn, err := grpc.NewClient(fmt.Sprintf("%v:%v", link.Hostname(), port), grpc.WithTransportCredentials(tls))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to printing server: %w", err)
	}

	return printerpb.NewPrinterConnectorClient(conn), nil
}
