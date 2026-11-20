package skills

import (
	"context"

	agent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
)

type AgentChatService struct {
	client agent.SessionService
}

func NewAgentChatService(client agent.SessionService) *AgentChatService {
	return &AgentChatService{client: client}
}

func (s *AgentChatService) Send(ctx context.Context, sessionUUID, messageUUID, idempotencyKey string) (*agent.ServiceInvocation, error) {
	if s == nil || s.client == nil {
		return nil, &agent.Error{Code: "AGENT_SESSION_UNAVAILABLE", ReasonCode: "AGENT_SESSION_UNAVAILABLE", StatusCode: 503}
	}
	return s.client.InvokeSession(ctx, sessionUUID, messageUUID, idempotencyKey)
}
