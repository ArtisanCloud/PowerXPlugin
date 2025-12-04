package templates

import (
	"embed"
)

//go:embed data/*
//go:embed data/**/*
//go:embed data/web-admin/nuxt/app/composables/api/_base.ts.tmpl
//go:embed data/web-admin/nuxt/app/composables/api/_client.ts.tmpl
//go:embed data/backend/go-gin/.env.example.tmpl
//go:embed data/web-admin/nuxt/.env.example.tmpl
var content embed.FS
