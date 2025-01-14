package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	printerpb "github.com/eolymp/go-sdk/eolymp/printer"
	"github.com/eolymp/printer-agent/pkg/ipp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func ConnectToServer() (printerpb.PrinterConnectorClient, error) {
	link, err := url.Parse(config.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL %q: %v", config.ServerURL, err)
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

func WatchPrinterState(ctx context.Context, printer *ipp.Client, state chan<- ipp.PrinterState) func() error {
	return func() error {
		for {
			attrs, err := printer.PrinterAttributes(ctx)
			if err != nil {
				return fmt.Errorf("failed to get printer attributes: %w", err)
			}

			// send printer state
			select {
			case state <- attrs.State:
			case <-ctx.Done():
				return nil
			}

			// wait for 3 seconds before next check
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func WatchJobState(ctx context.Context, printer *ipp.Client, job int, state chan<- ipp.JobState) func() error {
	return func() error {
		prev := ipp.JobPending

		for {
			attrs, err := printer.JobAttributes(ctx, job)
			if err != nil {
				return fmt.Errorf("failed to get job attributes: %w", err)
			}

			if prev != attrs.State {
				select {
				case state <- attrs.State:
					prev = attrs.State
				case <-ctx.Done():
					return nil
				}
			}

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return nil
			}
		}
	}
}
