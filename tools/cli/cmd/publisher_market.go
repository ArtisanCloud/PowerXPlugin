package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/powerx-plugin/cli/internal/publisher"
)

func executePublisherMarket(ctx context.Context, args []string, out io.Writer, locale string) error {
	if len(args) < 2 {
		return publisherInput("command")
	}
	group, action := args[0], args[1]
	flags := flag.NewFlagSet("publisher", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	entry := flags.String("entry", ".", publisher.Text(locale, "flag.entry"))
	idKey := "flag.connection_id"
	if group == "upload" {
		idKey = "flag.upload_id"
	}
	if group == "review" && action != "history" {
		idKey = "flag.submission_id"
	}
	id := flags.String("id", "", publisher.Text(locale, idKey))
	var input publisher.MarketInput
	var upload publisher.UploadInput
	var stdin bool
	var page, size int
	var connectionID, versionID, requestFile string
	switch group + "-" + action {
	case "market-add":
		flags.StringVar(&input.Name, "name", "", publisher.Text(locale, "flag.market_name"))
		flags.StringVar(&input.APIRoot, "api-root", "", publisher.Text(locale, "flag.api_root"))
		flags.StringVar(&input.ConsoleBase, "console-base", "", publisher.Text(locale, "flag.console_base"))
	case "market-bind":
		flags.BoolVar(&stdin, "token-stdin", false, publisher.Text(locale, "flag.token_stdin"))
	case "market-versions", "review-history", "review-requests", "upload-requests":
		flags.IntVar(&page, "page", 1, publisher.Text(locale, "flag.page"))
		flags.IntVar(&size, "size", 20, publisher.Text(locale, "flag.size"))
	case "review-prepare":
		flags.StringVar(&connectionID, "connection-id", "", publisher.Text(locale, "flag.connection_id"))
		flags.StringVar(&versionID, "version-id", "", publisher.Text(locale, "flag.version_id"))
	case "review-preview", "review-submit":
		flags.StringVar(&requestFile, "request-file", "", publisher.Text(locale, "flag.request_file"))
	case "review-list", "review-get", "review-resume", "review-cancel":
	case "market-list", "market-get", "market-verify", "market-disconnect", "upload-list", "upload-get", "upload-resume", "upload-cancel":
	case "upload-run":
		flags.StringVar(&upload.ConnectionID, "connection-id", "", publisher.Text(locale, "flag.connection_id"))
		flags.StringVar(&upload.BuildID, "build-id", "", publisher.Text(locale, "flag.build_id"))
	default:
		return publisherInput("command")
	}
	if flags.Parse(args[2:]) != nil || flags.NArg() != 0 {
		return publisherInput("arguments")
	}
	engine, err := publisher.Open(*entry)
	if err != nil {
		return err
	}
	defer engine.Close()
	var value any
	switch group + "-" + action {
	case "review-requests", "upload-requests":
		kind := "submissions"
		if group == "upload" {
			kind = "uploads"
		}
		value, err = engine.RequestTraces(kind, *id, page, size)
	case "review-prepare":
		value, err = engine.PrepareSubmission(ctx, connectionID, versionID)
	case "review-history":
		value, err = engine.MarketPublications(ctx, *id, page, size)
	case "review-list":
		value, err = engine.ListSubmissions()
	case "review-get":
		value, err = engine.GetSubmission(*id)
	case "review-cancel":
		value, err = engine.CancelSubmission(*id)
	case "review-preview", "review-submit":
		f, openErr := os.Open(requestFile)
		if openErr != nil {
			return publisherInput("request_file")
		}
		raw, readErr := io.ReadAll(io.LimitReader(f, publisher.MaxRequestBytes+1))
		f.Close()
		if readErr != nil || len(raw) > publisher.MaxRequestBytes {
			return publisherInput("request_file")
		}
		input, decodeErr := publisher.DecodeSubmission(raw)
		if decodeErr != nil {
			return decodeErr
		}
		if action == "preview" {
			value, err = engine.PreviewSubmission(input)
		} else {
			s, startErr := engine.StartSubmission(ctx, input)
			err = startErr
			if err == nil {
				s, err = engine.WaitSubmission(s.SubmissionID)
				value = s
				if err == nil && s.Stage != "succeeded" {
					err = s.Error
				}
			}
		}
	case "review-resume":
		var s publisher.Submission
		s, err = engine.ResumeSubmission(ctx, *id)
		if err == nil {
			s, err = engine.WaitSubmission(s.SubmissionID)
			value = s
			if err == nil && s.Stage != "succeeded" {
				err = s.Error
			}
		}
	case "market-add":
		value, err = engine.CreateMarket(input)
	case "market-list":
		value, err = engine.ListMarkets()
	case "market-get":
		value, err = engine.GetMarket(*id)
	case "market-bind":
		if !stdin {
			return publisherInput("token_stdin")
		}
		var b []byte
		b, err = io.ReadAll(io.LimitReader(os.Stdin, 52))
		if err == nil {
			if len(b) > 51 {
				return publisherInput("token")
			}
			value, err = engine.BindMarket(ctx, *id, strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r"))
		}
	case "market-verify":
		value, err = engine.VerifyMarket(ctx, *id)
	case "market-disconnect":
		value, err = engine.DisconnectMarket(ctx, *id)
	case "market-versions":
		value, err = engine.MarketVersions(ctx, *id, page, size)
	case "upload-list":
		value, err = engine.ListUploads()
	case "upload-get":
		value, err = engine.GetUpload(*id)
	case "upload-cancel":
		value, err = engine.CancelUpload(*id)
	case "upload-run", "upload-resume":
		var u publisher.Upload
		if action == "run" {
			u, err = engine.StartUpload(ctx, upload)
		} else {
			u, err = engine.ResumeUpload(ctx, *id)
		}
		if err == nil {
			u, err = engine.WaitUpload(u.UploadID)
		}
		value = u
		if err == nil && u.Stage != "succeeded" {
			err = u.Error
		}
	}
	if value != nil {
		if writeErr := json.NewEncoder(out).Encode(struct {
			Data any `json:"data"`
		}{value}); writeErr != nil {
			return writeErr
		}
	}
	return err
}
