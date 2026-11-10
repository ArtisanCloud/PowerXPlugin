package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/powerx-plugin/cli/internal/publisher"
)

func publisherLocale() string {
	if value := os.Getenv("LC_ALL"); value != "" {
		return value
	}
	return os.Getenv("LANG")
}
func runPublisher(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := executePublisher(ctx, args, os.Stdout, publisherLocale()); err != nil {
		return &publisher.DisplayError{Err: err, Locale: publisherLocale()}
	}
	return nil
}
func publisherInput(field string) error {
	return &publisher.Error{Code: publisher.InvalidInput, MessageKey: publisher.InvalidInput, Params: map[string]string{"field": field}}
}

// Both HTTP and CLI consume publisher.Engine. No market credentials, legacy
// configuration, arbitrary commands, or shell strings are accepted here.
func executePublisher(ctx context.Context, args []string, out io.Writer, locale string) error {
	if len(args) == 0 || len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(out, publisher.Text(locale, "usage"))
		return err
	}
	command := ""
	if args[0] == "serve" {
		command = "serve"
		args = args[1:]
	} else if len(args) >= 2 && args[0] == "plan" {
		command = args[1]
		args = args[2:]
		if command != "create" && command != "list" && command != "get" {
			return publisherInput("command")
		}
	} else if len(args) >= 2 && args[0] == "build" {
		action := args[1]
		if action != "run" && action != "get" && action != "list" && action != "verify" && action != "logs" {
			return publisherInput("command")
		}
		command = "build-" + action
		args = args[2:]
	} else {
		return publisherInput("command")
	}
	flags := flag.NewFlagSet("publisher", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	entry := flags.String("entry", ".", publisher.Text(locale, "flag.entry"))
	var input publisher.CreatePlan
	var notesFile, id, listen, origin, planID string
	page, size := 1, publisher.DefaultPageSize
	switch command {
	case "create":
		flags.StringVar(&input.Version, "version", "", publisher.Text(locale, "flag.version"))
		flags.StringVar(&input.Channel, "channel", "stable", publisher.Text(locale, "flag.channel"))
		flags.StringVar(&input.Target.OS, "os", "linux", publisher.Text(locale, "flag.os"))
		flags.StringVar(&input.Target.Arch, "arch", "amd64", publisher.Text(locale, "flag.arch"))
		flags.StringVar(&notesFile, "notes-file", "", publisher.Text(locale, "flag.notes"))
	case "get":
		flags.StringVar(&id, "id", "", publisher.Text(locale, "flag.id"))
	case "list", "build-list":
		flags.IntVar(&page, "page", 1, publisher.Text(locale, "flag.page"))
		flags.IntVar(&size, "size", publisher.DefaultPageSize, publisher.Text(locale, "flag.size"))
	case "serve":
		flags.StringVar(&listen, "listen", "127.0.0.1:0", publisher.Text(locale, "flag.listen"))
		flags.StringVar(&origin, "origin", "", publisher.Text(locale, "flag.origin"))
	case "build-run":
		flags.StringVar(&planID, "plan-id", "", publisher.Text(locale, "flag.plan_id"))
	case "build-get", "build-verify", "build-logs":
		flags.StringVar(&id, "id", "", publisher.Text(locale, "flag.build_id"))
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(out, publisher.Text(locale, "usage"))
			return err
		}
		return publisherInput("arguments")
	}
	if flags.NArg() != 0 {
		return publisherInput("arguments")
	}
	if notesFile != "" {
		f, err := os.Open(notesFile)
		if err != nil {
			return publisherInput("release_notes")
		}
		data, err := io.ReadAll(io.LimitReader(f, publisher.MaxNotesBytes+1))
		_ = f.Close()
		if err != nil || len(data) > publisher.MaxNotesBytes {
			return publisherInput("release_notes")
		}
		input.ReleaseNotes = string(data)
	}
	if command == "create" {
		if err := input.Validate(); err != nil {
			return err
		}
	}
	engine, err := publisher.Open(*entry)
	if err != nil {
		return err
	}
	defer engine.Close()
	write := func(value any) error {
		return json.NewEncoder(out).Encode(struct {
			Data any `json:"data"`
		}{value})
	}
	switch command {
	case "create":
		plan, err := engine.Create(input)
		if err != nil {
			return err
		}
		return write(plan)
	case "get":
		plan, err := engine.Get(id)
		if err != nil {
			return err
		}
		return write(plan)
	case "list":
		plans, err := engine.List(page, size)
		if err != nil {
			return err
		}
		return write(plans)
	case "serve":
		server, err := publisher.Start(ctx, engine, publisher.ServerOptions{Listen: listen, Origin: origin})
		if err != nil {
			return err
		}
		defer server.Close()
		if err = write(server.Info()); err != nil {
			return err
		}
		return server.Wait()
	case "build-run":
		b, err := engine.StartBuild(ctx, planID)
		if err != nil {
			return err
		}
		b, err = engine.WaitBuild(b.BuildID)
		if err != nil {
			return err
		}
		if err = write(b); err != nil {
			return err
		}
		if b.Stage != "ready" {
			return b.Error
		}
		return nil
	case "build-get":
		b, err := engine.GetBuild(id)
		if err != nil {
			return err
		}
		return write(b)
	case "build-list":
		b, err := engine.ListBuilds(page, size)
		if err != nil {
			return err
		}
		return write(b)
	case "build-verify":
		a, err := engine.VerifyArtifact(id)
		if err != nil {
			return err
		}
		return write(a)
	case "build-logs":
		s, err := engine.BuildLog(id)
		if err != nil {
			return err
		}
		return write(struct {
			Text string `json:"text"`
		}{s})
	}
	return publisherInput("command")
}
