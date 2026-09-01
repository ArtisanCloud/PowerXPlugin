package iamcontext

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/google/uuid"
)

// IdentityContextResolver 统一解析 bearer token 中的身份上下文。
type IdentityContextResolver struct{}

// ResolveIdentity 解析 tenant/user/roles/permissions 等统一字段。
func (IdentityContextResolver) ResolveIdentity(_ context.Context, bearerToken string) (*contracts.IdentityContext, error) {
	token := normalizeBearerToken(bearerToken)
	if token == "" {
		return nil, iamerrors.New(iamerrors.CodeUnauthorized, "bearer token is required")
	}

	claims, err := decodeJWTClaims(token)
	if err != nil {
		return nil, iamerrors.Wrap(iamerrors.CodeUnauthorized, "invalid bearer token", err)
	}

	tenantUUID := firstClaimString(claims, "tid", "tenant_uuid")
	if tenantUUID == "" {
		return nil, iamerrors.New(iamerrors.CodeUnauthorized, "tenant claim is required")
	}
	parsedTenant, parseErr := uuid.Parse(tenantUUID)
	if parseErr != nil {
		return nil, iamerrors.Wrap(iamerrors.CodeUnauthorized, "tenant claim is invalid", parseErr)
	}

	return &contracts.IdentityContext{
		TenantUUID:  strings.ToLower(parsedTenant.String()),
		UserUUID:    firstClaimString(claims, "user_uuid"),
		MemberUUID:  firstClaimString(claims, "member_uuid"),
		Roles:       parseStringSliceClaim(claims["roles"]),
		Permissions: parseStringSliceClaim(claims["permissions"]),
		PolicyVer:   firstClaimString(claims, "policy_version", "policy_ver"),
		TraceID:     firstClaimString(claims, "trace_id", "trace", "jti"),
	}, nil
}

func normalizeBearerToken(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		value = strings.TrimSpace(value[7:])
	}
	return value
}

func decodeJWTClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("jwt token format invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func firstClaimString(claims map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := claims[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		case float64:
			if v == float64(int64(v)) {
				return strconv.FormatInt(int64(v), 10)
			}
			return strings.TrimSpace(fmt.Sprint(v))
		default:
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func parseStringSliceClaim(raw any) []string {
	normalized := make([]string, 0)
	seen := map[string]struct{}{}
	push := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	switch value := raw.(type) {
	case []string:
		for _, item := range value {
			push(item)
		}
	case []any:
		for _, item := range value {
			push(fmt.Sprint(item))
		}
	case string:
		if strings.Contains(value, ",") {
			for _, item := range strings.Split(value, ",") {
				push(item)
			}
		} else {
			push(value)
		}
	default:
		if value != nil {
			push(fmt.Sprint(value))
		}
	}
	return normalized
}
