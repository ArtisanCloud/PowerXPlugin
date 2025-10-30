package templates

import (
	"embed"
)

//go:embed data/*
//go:embed data/**/*
var content embed.FS
