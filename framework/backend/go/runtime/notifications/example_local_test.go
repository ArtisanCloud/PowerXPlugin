package notifications_test

import (
	"context"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	powerxnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Contract demonstration only. A real plugin implements persistence, trusted
// tenant scope and authorization here; Framework does not create that store.
type exampleLocalPublisher struct{}

var _ notifications.Publisher = (*exampleLocalPublisher)(nil)

func (*exampleLocalPublisher) Create(context.Context, powerxnotifications.CreateInput) (*powerxnotifications.Notification, error) {
	return &powerxnotifications.Notification{UUID: "11111111-1111-4111-8111-111111111111"}, nil
}

func ExampleNewRuntime() {
	runtime, err := notifications.NewRuntime(provider.ModeLocal, &exampleLocalPublisher{}, nil)
	if err != nil {
		panic(err)
	}
	publisher, err := runtime.Publisher()
	if err != nil {
		panic(err)
	}
	out, err := publisher.Create(context.Background(), powerxnotifications.CreateInput{})
	fmt.Println(out != nil && err == nil)

	missing, err := notifications.NewRuntime(provider.ModeDelegated, &exampleLocalPublisher{}, nil)
	if err != nil {
		panic(err)
	}
	_, err = missing.Publisher()
	fmt.Println(err != nil)
	// Output:
	// true
	// true
}
