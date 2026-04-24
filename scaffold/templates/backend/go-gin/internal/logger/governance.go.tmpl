package logger

import (
	"strconv"
	"strings"
)

type GovernanceStatus string

const (
	GovernanceDetected GovernanceStatus = "detected"
	GovernanceWarned   GovernanceStatus = "warned"
	GovernanceBlocked  GovernanceStatus = "blocked"
	GovernanceResolved GovernanceStatus = "resolved"
)

type GovernancePolicy struct {
	Mode            string
	DeadlineVersion string
	CurrentVersion  string
}

func (p GovernancePolicy) Status(violationCount int) GovernanceStatus {
	if violationCount <= 0 {
		return GovernanceResolved
	}
	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	switch mode {
	case "detect":
		return GovernanceDetected
	case "block":
		return GovernanceBlocked
	case "warn", "":
		return GovernanceWarned
	default:
		return GovernanceWarned
	}
}

func (p GovernancePolicy) ShouldBlockByDeadline() bool {
	deadline := strings.TrimSpace(p.DeadlineVersion)
	current := strings.TrimSpace(p.CurrentVersion)
	if deadline == "" || current == "" {
		return false
	}
	return compareVersion(current, deadline) >= 0
}

func compareVersion(current, target string) int {
	curParts := normalizeVersionParts(current)
	tgtParts := normalizeVersionParts(target)
	maxLen := len(curParts)
	if len(tgtParts) > maxLen {
		maxLen = len(tgtParts)
	}
	for i := 0; i < maxLen; i++ {
		curVal := 0
		if i < len(curParts) {
			curVal = curParts[i]
		}
		tgtVal := 0
		if i < len(tgtParts) {
			tgtVal = tgtParts[i]
		}
		if curVal > tgtVal {
			return 1
		}
		if curVal < tgtVal {
			return -1
		}
	}
	return 0
}

func normalizeVersionParts(raw string) []int {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(raw), "v"))
	if trimmed == "" {
		return []int{0}
	}
	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	parts := make([]int, 0, len(segments))
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		numericPrefix := seg
		for i, ch := range seg {
			if ch < '0' || ch > '9' {
				numericPrefix = seg[:i]
				break
			}
		}
		if numericPrefix == "" {
			parts = append(parts, 0)
			continue
		}
		value, err := strconv.Atoi(numericPrefix)
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return []int{0}
	}
	return parts
}
