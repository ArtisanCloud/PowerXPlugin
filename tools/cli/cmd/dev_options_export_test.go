package cmd

// ExportTokenTest is a helper for integration/run to inspect token resolution.
func ExportTokenTest(devAPIBase string) string {
	return resolveDevAPIToken(&DevOptions{}, devAPIBase)
}
