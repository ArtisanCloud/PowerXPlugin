package manifest

import (
	"gopkg.in/yaml.v3"
	"os"
)

func LoadNotificationPresentations(path string) ([]NotificationPresentation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	return DecodeNotificationPresentations(document["notification_presentations"])
}
