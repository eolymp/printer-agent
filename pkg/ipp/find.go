package ipp

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

type Printer struct {
	Name  string
	State PrinterState
	URI   string
}

func Find(ctx context.Context) (chan *Printer, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	printers := make(chan *Printer)
	discovery := make(chan *zeroconf.ServiceEntry)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(printers)
				return
			case entry := <-discovery:
				printers <- ParseFindEntry(entry)
			}
		}
	}()

	if err := resolver.Browse(ctx, "_ipp._tcp", "local.", discovery); err != nil {
		return nil, fmt.Errorf("failed to browse: %w", err)
	}

	return printers, nil
}

func ParseFindEntry(entry *zeroconf.ServiceEntry) *Printer {
	attr := map[string]string{}
	for _, txt := range entry.Text {
		kv := strings.SplitN(txt, "=", 2)
		attr[kv[0]] = kv[1]
	}

	state, _ := strconv.Atoi(attr["printer-state"])

	local, _ := os.Hostname()
	hostname := strings.TrimSuffix(entry.HostName, ".")

	if hostname == local {
		hostname = "localhost"
	}

	uri := url.URL{
		Scheme: "ipp",
		Host:   fmt.Sprintf("%v:%v", hostname, strconv.Itoa(entry.Port)),
		Path:   attr["rp"],
	}

	return &Printer{
		Name:  entry.Instance,
		State: PrinterState(state),
		URI:   uri.String(),
	}
}
