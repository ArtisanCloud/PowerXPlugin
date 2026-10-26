package cmd

import fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"

func loadNotificationPresentationsOption(path string) ([]fwmanifest.NotificationPresentation, error) {
	if path == "" {
		return nil, nil
	}
	return fwmanifest.LoadNotificationPresentations(path)
}
