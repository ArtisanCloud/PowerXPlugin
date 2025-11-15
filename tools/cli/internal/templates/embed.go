package templates

import (
	"embed"
)

//go:embed data/*
//go:embed data/**/*
//go:embed data/web-admin/nuxt/app/composables/api/_base.ts.tmpl
//go:embed data/web-admin/nuxt/app/composables/api/_client.ts.tmpl
var content embed.FS
