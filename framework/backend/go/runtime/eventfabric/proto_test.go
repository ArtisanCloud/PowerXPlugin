package eventfabric

import (
	"testing"

	pb "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestProtoCarriesServiceIdentityAndReceiptFields(t *testing.T) {
	for _, message := range []protoreflect.Message{(&pb.PublishEventRequest{}).ProtoReflect(), (&pb.SubscribeRequest{}).ProtoReflect(), (&pb.AckDeliveryRequest{}).ProtoReflect(), (&pb.NackDeliveryRequest{}).ProtoReflect()} {
		for _, name := range []protoreflect.Name{"contract_version", "plugin_id", "tenant_uuid"} {
			require.NotNil(t, message.Descriptor().Fields().ByName(name))
		}
	}
	message := (&pb.DeliveryMessage{}).ProtoReflect()
	for _, name := range []protoreflect.Name{"contract_version", "tenant_uuid", "publisher_plugin_id", "published_by", "idempotency_key", "delivery_id", "ack_deadline"} {
		require.NotNil(t, message.Descriptor().Fields().ByName(name))
	}
}
