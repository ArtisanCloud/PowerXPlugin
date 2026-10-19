package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/google/uuid"
	Regression "github.com/powerx-plugin/cli/internal/regression"
	"io"
	"os"
	"strings"
)

type regressionCaseFlags []string

func (values *regressionCaseFlags) String() string { return strings.Join(*values, ",") }
func (values *regressionCaseFlags) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func runRegression(args []string) error {
	if len(args) == 0 {
		return &Regression.Error{Code: "REGRESSION_INVALID_REQUEST"}
	}
	flags := flag.NewFlagSet("regression", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	base := flags.String("api-base", "", "")
	tokenEnv := flags.String("token-env", "", "")
	tenant := flags.String("tenant", "", "")
	plugin := flags.String("plugin-id", "", "")
	locale := flags.String("locale", "zh", "")
	profile := flags.String("profile", "", "")
	var cases regressionCaseFlags
	flags.Var(&cases, "case", "")
	run := flags.String("run", "", "")
	wait := flags.Bool("wait", false, "")
	input := flags.String("inputs", "", "")
	request := flags.String("request-uuid", uuid.NewString(), "")
	report := flags.String("report", "", "")
	output := flags.String("output", "", "")
	format := flags.String("format", "json", "")
	_ = flags.Bool("json", false, "")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return &Regression.Error{Code: "REGRESSION_INVALID_REQUEST", Locale: *locale}
	}
	token := os.Getenv(*tokenEnv)
	if token == "" {
		return &Regression.Error{Code: "REGRESSION_INVALID_REQUEST", Locale: *locale}
	}
	c := &Regression.Client{BaseURL: *base, Token: token, TenantUUID: *tenant, PluginID: *plugin, Locale: *locale}
	ctx := context.Background()
	switch args[0] {
	case "list":
		b, err := c.List(ctx)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(b, '\n'))
		return err
	case "run":
		if *wait && *plugin == "" {
			return &Regression.Error{Code: "REGRESSION_INVALID_REQUEST", Locale: *locale}
		}
		inputs := json.RawMessage(`{}`)
		if *input != "" {
			b, err := os.ReadFile(*input)
			if err != nil {
				return err
			}
			if !json.Valid(b) {
				return &Regression.Error{Code: "REGRESSION_INVALID_REQUEST", Locale: *locale}
			}
			inputs = b
		}
		d, err := c.Create(ctx, *request, *profile, cases, inputs)
		if err != nil {
			return err
		}
		if b, e := json.Marshal(d); e == nil {
			_, _ = os.Stdout.Write(append(b, '\n'))
		}
		if *wait {
			d, err = c.Wait(ctx, d)
		}
		if *report != "" {
			b, reportErr := c.Report(ctx, d.Run.UUID, "json")
			if reportErr != nil {
				return reportErr
			}
			if writeErr := os.WriteFile(*report, b, 0600); writeErr != nil {
				return writeErr
			}
		}
		return err
	case "report":
		b, err := c.Report(ctx, *run, *format)
		if err != nil {
			return err
		}
		if *output != "" {
			return os.WriteFile(*output, b, 0600)
		}
		_, err = os.Stdout.Write(b)
		return err
	default:
		return &Regression.Error{Code: "REGRESSION_INVALID_REQUEST", Locale: *locale}
	}
}
