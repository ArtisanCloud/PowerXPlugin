package manifest

import "testing"

func fixtureBindings() []NotificationPresentation {
	return []NotificationPresentation{
		{BusinessType: "sample_track", Contract: BusinessNotificationCardContract, PresentationPath: "/admin/operations/order-content-notifications/sample-tracks/{delivery_key}/presentation", PreviewPath: "/admin/operations/order-content-notifications/sample-tracks/{delivery_key}/images/{image_key}"},
		{BusinessType: "bulk_order", Contract: BusinessNotificationCardContract, PresentationPath: "/admin/operations/order-content-notifications/bulk-orders/{delivery_key}/presentation", PreviewPath: "/admin/operations/order-content-notifications/bulk-orders/{delivery_key}/images/{image_key}"},
	}
}
func TestNotificationPresentationRegistrationAndPathValidation(t *testing.T) {
	bindings := fixtureBindings()
	if err := Validate(Plugin{ID: "com.powerx.demo", Name: "fixture", NotificationPresentations: bindings}); err != nil {
		t.Fatal(err)
	}
	if ValidateNotificationPresentations(append(bindings, bindings[0])) == nil {
		t.Fatal("DUPLICATE_BUSINESS_TYPE_ACCEPTED")
	}
	for _, path := range []string{"https://external/admin/{delivery_key}", "/api/v1/admin/{delivery_key}", "/admin/{delivery_key}?a=1", "/admin/{delivery_key}#image", "/admin/../{delivery_key}", "/admin/%2e%2e/{delivery_key}", "/admin/%252e%252e/{delivery_key}", "/admin//{delivery_key}", "/admin/{unknown}/{delivery_key}", "/admin/{delivery_key}/{delivery_key}", "/admin/no-placeholder", "/admin/{delivery_key}/{image_key}", "/admin/{delivery_key}/", "/admin/\\{delivery_key}"} {
		binding := bindings[0]
		binding.PresentationPath = path
		if binding.Validate() == nil {
			t.Fatalf("INVALID_PATH_ACCEPTED:%s", path)
		}
	}
	for _, path := range []string{"/admin/{delivery_key}", "/admin/{image_key}/{image_key}/{delivery_key}", "/admin/{delivery_key}/{unknown}"} {
		binding := bindings[0]
		binding.PreviewPath = path
		if binding.Validate() == nil {
			t.Fatal("INVALID_PREVIEW_ACCEPTED")
		}
	}
	if _, err := DecodeNotificationPresentations([]map[string]any{{"unknown": true}}); err == nil {
		t.Fatal("UNKNOWN_BINDING_FIELD_ACCEPTED")
	}
}

func TestNotificationPresentationVersionsCoexist(t *testing.T) {
	bindings := fixtureBindings()
	v2 := bindings[0]
	v2.Contract = BusinessNotificationCardContractV2
	bindings = append(bindings, v2)
	if err := Validate(Plugin{ID: "com.powerx.demo", Name: "fixture", NotificationPresentations: bindings}); err != nil {
		t.Fatal(err)
	}
	if ValidateNotificationPresentations(append(bindings, v2)) == nil {
		t.Fatal("SAME_VERSION_DUPLICATE_ACCEPTED")
	}
	invalid := v2
	invalid.Contract = "business_notification_card.v3"
	if invalid.Validate() == nil {
		t.Fatal("UNKNOWN_CONTRACT_ACCEPTED")
	}
}
