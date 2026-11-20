package runtimeexample

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	powerxruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginruntime"
	"github.com/google/uuid"
)

// LocalPluginRuntime is the startup-bound local implementation of the Plugin
// Runtime contract. Its records are tenant-scoped and process-local, matching
// the other local development adapters. It never calls a Core endpoint.
type LocalPluginRuntime struct {
	ai        *LocalAI
	knowledge fwknowledge.KnowledgeProvider
	mu        sync.RWMutex
	agents    map[string]map[string]localPluginRuntimeAgent
}

type localPluginRuntimeAgent struct {
	value    powerxruntime.Agent
	modelKey string
}

func NewLocalPluginRuntime(ai *LocalAI, knowledge fwknowledge.KnowledgeProvider) (*LocalPluginRuntime, error) {
	if ai == nil || knowledge == nil {
		return nil, agentError(400, "LOCAL_CONFIG_INVALID")
	}
	return &LocalPluginRuntime{ai: ai, knowledge: knowledge, agents: map[string]map[string]localPluginRuntimeAgent{}}, nil
}

func (s *LocalPluginRuntime) ListKnowledgeSpaces(ctx context.Context, in powerxruntime.ListKnowledgeSpacesInput) (*powerxruntime.ListKnowledgeSpacesOutput, error) {
	tenantUUID, err := agentScope(ctx)
	if err != nil {
		return nil, err
	}
	if in.Page < 1 || in.PageSize < 1 || in.PageSize > fwknowledge.MaxQueryLimit {
		return nil, agentError(400, "INVALID_ARGUMENT")
	}
	spaces, err := s.knowledge.ListSpaces(ctx, fwknowledge.ListSpacesInput{TenantUUID: tenantUUID, Status: strings.TrimSpace(in.Status), Limit: fwknowledge.MaxQueryLimit})
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	items := make([]powerxruntime.KnowledgeSpace, 0, len(spaces))
	for _, space := range spaces {
		if keyword != "" && !strings.Contains(strings.ToLower(space.SpaceName), keyword) && !strings.Contains(strings.ToLower(space.SpaceID), keyword) {
			continue
		}
		items = append(items, powerxruntime.KnowledgeSpace{UUID: space.SpaceID, SpaceName: space.SpaceName, Status: space.Status, DepartmentCode: space.DepartmentCode, RAGProfileKey: space.RAGProfileKey})
	}
	start := (in.Page - 1) * in.PageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + in.PageSize
	if end > len(items) {
		end = len(items)
	}
	return &powerxruntime.ListKnowledgeSpacesOutput{Items: items[start:end], Total: int64(len(items)), Page: in.Page, PageSize: in.PageSize}, nil
}

func (s *LocalPluginRuntime) ListAgents(ctx context.Context, environment, status string) ([]powerxruntime.Agent, error) {
	tenantUUID, err := agentScope(ctx)
	if err != nil {
		return nil, err
	}
	environment, status = strings.TrimSpace(environment), strings.TrimSpace(status)
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]powerxruntime.Agent, 0, len(s.agents[tenantUUID]))
	for _, record := range s.agents[tenantUUID] {
		if environment != "" && record.value.Environment != environment || status != "" && record.value.Status != status {
			continue
		}
		items = append(items, clonePluginRuntimeAgent(record.value))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UUID < items[j].UUID })
	return items, nil
}

func (s *LocalPluginRuntime) InstantiateAgent(ctx context.Context, in powerxruntime.InstantiateAgentInput) (*powerxruntime.Agent, error) {
	tenantUUID, err := agentScope(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 256 || len(in.SkillIDs) > 128 || len(in.KnowledgeBaseIDs) > 128 {
		return nil, agentError(400, "INVALID_ARGUMENT")
	}
	for _, id := range append(append([]string(nil), in.SkillIDs...), in.KnowledgeBaseIDs...) {
		if parsed, parseErr := uuid.Parse(id); parseErr != nil || parsed == uuid.Nil {
			return nil, agentError(400, "INVALID_ARGUMENT")
		}
	}
	modelKey, ok := in.Parameters["model_key"].(string)
	modelKey = strings.TrimSpace(modelKey)
	if !ok || modelKey == "" {
		return nil, agentError(400, "MODEL_KEY_REQUIRED")
	}
	if _, exists := s.ai.models[modelKey]; !exists {
		return nil, agentError(404, "MODEL_NOT_FOUND")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	value := powerxruntime.Agent{UUID: uuid.NewString(), Key: strings.TrimSpace(in.Key), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), Persona: strings.TrimSpace(in.Persona), Environment: firstRuntimeValue(in.Environment, "local"), Status: firstRuntimeValue(in.Status, "active"), Visibility: firstRuntimeValue(in.Visibility, "tenant"), Scope: firstRuntimeValue(in.Scope, "tenant"), Source: "local", TypeID: strings.TrimSpace(in.TypeID), Scene: strings.TrimSpace(in.Scene), PromptSeed: strings.TrimSpace(in.PromptSeed), Parameters: cloneMap(in.Parameters), SkillIDs: append([]string(nil), in.SkillIDs...), KnowledgeBaseIDs: append([]string(nil), in.KnowledgeBaseIDs...), Meta: cloneMap(in.Meta), CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agents[tenantUUID] == nil {
		s.agents[tenantUUID] = map[string]localPluginRuntimeAgent{}
	}
	s.agents[tenantUUID][value.UUID] = localPluginRuntimeAgent{value: value, modelKey: modelKey}
	out := clonePluginRuntimeAgent(value)
	return &out, nil
}

func firstRuntimeValue(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func clonePluginRuntimeAgent(value powerxruntime.Agent) powerxruntime.Agent {
	value.Parameters = cloneMap(value.Parameters)
	value.Meta = cloneMap(value.Meta)
	value.SkillIDs = append([]string(nil), value.SkillIDs...)
	value.KnowledgeBaseIDs = append([]string(nil), value.KnowledgeBaseIDs...)
	return value
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}
