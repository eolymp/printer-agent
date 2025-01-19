package main

import (
	"context"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	printerpb "github.com/eolymp/go-sdk/eolymp/printer"
	"github.com/eolymp/printer-agent/pkg/connector"
	"github.com/eolymp/printer-agent/pkg/ipp"
	"github.com/eolymp/printer-agent/pkg/messages"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
)

var version = "0.0.0"
var commit = "HEAD"

var config struct {
	PrinterURL    string
	ServerURL     string
	Space         string
	Token         string
	JobTTL        time.Duration
	LookupTimeout time.Duration
}

func main() {
	ctx := context.Background()

	flag.Usage = func() {
		f := flag.CommandLine.Output()
		_, _ = fmt.Fprintf(f, "Usage: %s [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(f, "Version: %s (%s)\n", version, commit)
		_, _ = fmt.Fprintln(f, "Options:")
		flag.PrintDefaults()
		_, _ = fmt.Fprintln(f, "")
		_, _ = fmt.Fprintln(f, "Learn more: https://github.com/eolymp/printer-agent/blob/main/README.md")
	}

	flag.StringVar(&config.PrinterURL, "printer", "", "Printer URL. Use \"ipps\" schema to connect with TLS, add username and password to authenticate.")
	flag.StringVar(&config.ServerURL, "server", "https://printer.eolymp.com", "Server hostname")
	flag.StringVar(&config.Space, "space", "", "Space ID where printer is hosted")
	flag.StringVar(&config.Token, "token", "", "Server token to authenticate printer")
	flag.DurationVar(&config.JobTTL, "job-ttl", 5*time.Minute, "Job time-to-live, if job is older it will be cancelled")
	flag.DurationVar(&config.LookupTimeout, "lookup-timeout", time.Minute, "A timeout for printer lookup")
	flag.Parse()

	// validate config
	if config.PrinterURL == "" {
		f := flag.CommandLine.Output()
		_, _ = fmt.Fprintln(f, "You should provide -printer option to connect to the printer and start printing documents.")
		_, _ = fmt.Fprintln(f, "Learn more: https://github.com/eolymp/printer-agent/blob/main/README.md")
		_, _ = fmt.Fprintln(f, "")
		_, _ = fmt.Fprintln(f, "Looking for available printers...")

		ctx, cancel := context.WithTimeout(ctx, config.LookupTimeout)
		defer cancel()

		printers, err := ipp.Find(ctx)
		if err != nil {
			fmt.Fprintf(f, "Failed to find printers: %v\n", err)
		}

		count := 0
		for printer := range printers {
			fmt.Fprintf(f, "- %s (%s)\n", printer.URI, printer.State)
			count++
		}

		if count == 0 {
			_, _ = fmt.Fprintln(f, "No printers found.")
		} else {
			_, _ = fmt.Fprintln(f, "")
			_, _ = fmt.Fprintln(f, "Please, restart this application with -printer option.")
		}

		os.Exit(0)
	}

	if config.Space == "" {
		fail("Space ID is required, please add -space argument")
	}

	if config.Token == "" {
		fail("Printer token is required, please add -token argument")
	}

	// connect to printer
	printer := ipp.New(config.PrinterURL)

	// connect to printing server
	cli, err := connector.Connect(config.ServerURL)
	if err != nil {
		fail("Failed to connect to printing server: %v", err)
	}

	// run printing agent
	attempt := time.Now()
	backoff := time.Second / 2

	for {
		attempt = time.Now()

		if err := run(ctx, cli, printer); err != nil {
			// this is a known error happening due to server IDLE timeout, ignore it
			if strings.Contains(err.Error(), "stream terminated by RST_STREAM with error code: PROTOCOL_ERROR") {
				log.Println("Connection closed due to inactivity, reconnecting...")
				continue
			}

			log.Printf("ERROR: %v", err)

			backoff = backoff * 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}

		// connection was maintained for at least 5 seconds, reconnect right away
		if time.Since(attempt) > 5*time.Second {
			backoff = time.Second / 2
			continue
		}

		log.Printf("Reconnecting in %v...", backoff)
		time.Sleep(backoff)
	}
}

func fail(msg string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR: "+msg+"\n", args...)
	flag.Usage()
	os.Exit(-1)
}

func run(ctx context.Context, cli printerpb.PrinterConnectorClient, printer *ipp.Client) error {
	stream, err := cli.Connect(metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+url.PathEscape(config.Token),
		"space-id", config.Space,
	))

	if err != nil {
		return fmt.Errorf("failed to connect to printing server: %w", err)
	}

	defer func() {
		_ = stream.CloseSend()
	}()

	// wait for hello message before starting the routine
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	if _, ok := msg.GetMessage().(*printerpb.PrinterConnectorServerMessage_Hello_); !ok {
		return fmt.Errorf("unexpected message: %T", msg)
	}

	log.Println("Connected to the server")

	// create an error group to manage routines
	eg, ctx := errgroup.WithContext(ctx)

	// start a routine to receive messages from the server
	msgs := make(chan *printerpb.PrinterConnectorServerMessage)
	eg.Go(func() error {
		for {
			msg, err := stream.Recv()
			if err != nil {
				return fmt.Errorf("failed to receive response: %w", err)
			}

			select {
			case msgs <- msg:
			case <-ctx.Done():
				return nil
			}
		}
	})

	// start a routine watching printer state
	state := make(chan ipp.PrinterState)
	eg.Go(printer.WatchPrinterState(ctx, state))

	// start a control flow routine
	eg.Go(func() error {
		printerState := ipp.PrinterStopped

		for {
			select {
			case msg := <-msgs:
				switch m := msg.GetMessage().(type) {
				case *printerpb.PrinterConnectorServerMessage_Print_:
					job := m.Print.GetJob()

					log.Printf("Received print job #%v: %v", job.GetId(), job.GetDocumentUrl())

					if since := time.Since(job.GetCreatedAt().AsTime()); since > config.JobTTL {
						log.Printf("Job is too old (created %v ago), skipping", since)

						if err := stream.Send(messages.Report(printerpb.Job_CANCELLED)); err != nil {
							return fmt.Errorf("failed to report job status: %w", err)
						}

						continue
					}

					filename, mime, err := download(ctx, job.GetDocumentUrl())
					if err != nil {
						return fmt.Errorf("failed to download document: %w", err)
					}

					num, err := printer.PrintJobFile(ctx, filename, mime)
					if err != nil {
						return fmt.Errorf("failed to print document: %w", err)
					}

					log.Printf("Printing job #%v added to the queue under number %v", job.GetId(), num)

					// mark job as completed right away
					if err := stream.Send(messages.Report(printerpb.Job_COMPLETE)); err != nil {
						return fmt.Errorf("failed to report job status: %w", err)
					}
				}

			case s := <-state:
				// do not report the same state
				if printerState == s {
					continue
				}

				switch s {
				case ipp.PrinterIdle:
					log.Println("The printer is ready")
					_ = stream.Send(messages.Status(printerpb.Printer_READY))
				case ipp.PrinterProcessing:
					log.Println("The printer is busy")
					_ = stream.Send(messages.Status(printerpb.Printer_BUSY))
				case ipp.PrinterStopped:
					log.Println("The printer is offline")
					_ = stream.Send(messages.Status(printerpb.Printer_OFFLINE))
				default:
					continue
				}

				printerState = s

			case <-ctx.Done():
				return nil
			}
		}
	})

	if err := eg.Wait(); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}

	return nil
}

func download(ctx context.Context, link string) (string, string, error) {
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%x", md5.Sum([]byte(link))))

	log.Printf("Downloading document %q to %q", link, filename)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to download document: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}

	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}

	return filename, resp.Header.Get("Content-Type"), nil
}
