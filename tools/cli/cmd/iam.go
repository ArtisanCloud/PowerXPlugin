package cmd

import "fmt"

// runIAM dispatches IAM management subcommands.
func runIAM(args []string) error {
	if len(args) == 0 {
		printIAMHelp()
		return nil
	}

	sub := args[0]
	switch sub {
	case "help", "-h", "--help":
		printIAMHelp()
		return nil
	case "export":
		return runIAMExport(args[1:])
	case "seed":
		return runIAMSeed(args[1:])
	default:
		return fmt.Errorf("unknown iam subcommand: %s", sub)
	}
}

func printIAMHelp() {
	fmt.Print(`px-plugin iam <command> [flags]

Manage local IAM data for Standalone mode directly from the px-plugin CLI.

Commands:
  iam export   Dump tenants/roles/members/permissions to JSON (for backup/migration)
  iam seed     Reset or bootstrap the default local tenant administrator

Examples:
  px-plugin iam export --entry ./skeleton --output /tmp/iam.json
  px-plugin iam seed --entry ./skeleton --admin-email admin@local.test --admin-password S3cret!!
`)
}
