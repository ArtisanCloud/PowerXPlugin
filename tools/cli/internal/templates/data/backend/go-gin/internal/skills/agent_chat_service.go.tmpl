package skills

import (
	"context"

	agent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
)

type AgentChatService struct {
	client *agent.Client
}

func NewAgentChatService(client *agent.Client) *AgentChatService {
	return &AgentChatService{client: client}
}

func (s *AgentChatService) Send(ctx context.Context, req agent.AgentInvokeRequest) (agent.AgentInvokeResponse, error) {
	if s == nil || s.client == nil {
		return agent.AgentInvokeResponse{}, &agent.Error{Code: agent.ErrCodeConfigInvalid, Message: "agent client is not configured"}
	}
	return s.client.Invoke(ctx, req)
}
