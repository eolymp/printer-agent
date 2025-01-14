package ipp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/OpenPrinting/goipp"
)

type Client struct {
	uri      string
	username string
	charset  string
	language string
}

func New(uri string) *Client {
	return &Client{
		uri:      uri,
		username: "IPP Library",
		charset:  "utf-8",
		language: "en-US",
	}
}

func (c *Client) PrinterAttributes(ctx context.Context) (*PrinterAttributes, error) {
	in := goipp.NewRequest(goipp.DefaultVersion, goipp.OpGetPrinterAttributes, 1)
	in.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String(c.charset)))
	in.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String(c.language)))
	in.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(c.uri)))
	in.Operation.Add(goipp.MakeAttribute("requested-attributes", goipp.TagKeyword, goipp.String("all")))

	out, err := c.SendRequest(ctx, in, nil)
	if err != nil {
		return nil, err
	}

	attrs := &PrinterAttributes{}

	for _, attr := range out.Printer {
		if len(attr.Values) == 0 {
			continue
		}

		switch attr.Name {
		case "printer-name":
			attrs.Name = string(attr.Values[0].V.(goipp.String))
		case "printer-info":
			attrs.Info = string(attr.Values[0].V.(goipp.String))
		case "printer-state":
			attrs.State = PrinterState(attr.Values[0].V.(goipp.Integer))
		case "printer-state-reasons":
			attrs.StateReason = string(attr.Values[0].V.(goipp.String))
		case "queued-job-count":
			attrs.QueuedJobCount = int(attr.Values[0].V.(goipp.Integer))
		case "color-supported":
			attrs.ColorSupported = bool(attr.Values[0].V.(goipp.Boolean))
		case "operations-supported":
			for _, v := range attr.Values {
				attrs.OperationsSupported = append(attrs.OperationsSupported, goipp.Op(v.V.(goipp.Integer)))
			}
		case "page-delivery-supported":
			for _, v := range attr.Values {
				attrs.PageDeliverySupported = append(attrs.PageDeliverySupported, string(v.V.(goipp.String)))
			}
		}
	}

	return attrs, nil
}

func (c *Client) JobAttributes(ctx context.Context, job int) (*JobAttributes, error) {
	in := goipp.NewRequest(goipp.DefaultVersion, goipp.OpGetJobAttributes, 1)
	in.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String(c.charset)))
	in.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String(c.language)))
	in.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(c.uri)))
	in.Operation.Add(goipp.MakeAttribute("job-id", goipp.TagInteger, goipp.Integer(job)))

	out, err := c.SendRequest(ctx, in, nil)
	if err != nil {
		return nil, err
	}

	attrs := &JobAttributes{}

	for _, attr := range out.Job {
		//fmt.Println(attr.Name)

		if len(attr.Values) == 0 {
			continue
		}

		switch attr.Name {
		case "job-state":
			attrs.State = JobState(attr.Values[0].V.(goipp.Integer))
		case "job-state-reasons":
			attrs.StateReason = string(attr.Values[0].V.(goipp.String))
		}
	}

	return attrs, nil
}

func (c *Client) PrintJob(ctx context.Context, filename, mime string, reader io.Reader) (int, error) {
	in := goipp.NewRequest(goipp.DefaultVersion, goipp.OpPrintJob, 1)
	in.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String(c.charset)))
	in.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String(c.language)))
	in.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(c.uri)))
	in.Operation.Add(goipp.MakeAttribute("requesting-user-name", goipp.TagName, goipp.String(c.username)))
	in.Operation.Add(goipp.MakeAttribute("job-name", goipp.TagName, goipp.String(filename)))
	in.Operation.Add(goipp.MakeAttribute("document-format", goipp.TagMimeType, goipp.String(mime)))

	out, err := c.SendRequest(ctx, in, reader)
	if err != nil {
		return -1, err
	}

	for _, attr := range out.Job {
		if len(attr.Values) == 0 {
			continue
		}

		switch attr.Name {
		case "job-id":
			return int(attr.Values[0].V.(goipp.Integer)), nil
		}
	}

	return -1, errors.New("no job-id in response")
}

func (c *Client) PrintJobFile(ctx context.Context, filename, mime string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return -1, fmt.Errorf("open file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	return c.PrintJob(ctx, filepath.Base(filename), mime, file)
}

func (c *Client) SendRequest(ctx context.Context, in *goipp.Message, payload io.Reader) (*goipp.Message, error) {
	message, err := in.EncodeBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	uri, err := url.Parse(c.uri)
	if err != nil {
		return nil, fmt.Errorf("invalid printer URI %q: %v", c.uri, err)
	}

	if uri.Scheme == "ipp" {
		uri.Scheme = "http"
	}

	if uri.Scheme == "ipps" {
		uri.Scheme = "https"
	}

	var body io.Reader = bytes.NewBuffer(message)
	if payload != nil {
		body = io.MultiReader(body, payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if uri.User != nil {
		pwd, _ := uri.User.Password()
		req.SetBasicAuth(uri.User.Username(), pwd)
	}

	req.Header.Set("Content-Type", goipp.ContentType)
	req.Header.Set("Accept", goipp.ContentType)
	req.Header.Set("Accept-Encoding", "gzip, deflate, identity")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("request failed: status code %v", resp.StatusCode)
	}

	out := goipp.Message{}
	if err := out.Decode(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if goipp.Status(out.Code) != goipp.StatusOk {
		return nil, errors.New(goipp.Status(out.Code).String())
	}

	return &out, nil
}
