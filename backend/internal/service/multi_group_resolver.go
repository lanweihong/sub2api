package service

import (
	"errors"
	"sort"
	"strings"
)

var (
	// ErrNoGroupMatchForModel is returned when no bound group matches the requested model.
	ErrNoGroupMatchForModel = errors.New("no group matches the requested model")
)

// ResolveGroupForModel selects the appropriate group for an API key based on the requested model.
//
// Resolution priority:
//  1. If the key has no bound groups and no legacy group_id, return nil (no group).
//  2. If the key has only the legacy group_id (no bound groups), return that group (backward compat).
//  3. If the key has exactly one bound group, return it directly.
//  4. If the key has multiple bound groups, match by model patterns and return the
//     highest-priority (lowest priority value) matching group.
//
// Model matching uses prefix patterns with wildcard support (e.g., "claude-*" matches "claude-opus-4").
// Per-binding model_patterns override the group's supported_model_scopes when present.
func ResolveGroupForModel(apiKey *APIKey, requestedModel string) (*Group, error) {
	// Case 1: No bound groups — use legacy group_id path
	if !apiKey.HasBoundGroups() {
		return apiKey.Group, nil // may be nil (no group assigned)
	}

	// Case 2: Single bound group — return directly (no model matching needed)
	if len(apiKey.BoundGroups) == 1 {
		bg := &apiKey.BoundGroups[0]
		if bg.Group == nil {
			return apiKey.Group, nil // fallback to legacy
		}
		return bg.Group, nil
	}

	// Case 3: Multiple bound groups — match by model
	if requestedModel == "" {
		// No model specified: return the highest priority group
		return apiKey.BoundGroups[0].Group, nil // already sorted by priority from DB
	}

	// Find all matching groups, sorted by priority (already sorted from DB query)
	for i := range apiKey.BoundGroups {
		bg := &apiKey.BoundGroups[i]
		if bg.Group == nil || !bg.Group.IsActive() {
			continue
		}
		if matchesGroup(bg, requestedModel) {
			return bg.Group, nil
		}
	}

	return nil, ErrNoGroupMatchForModel
}

// matchesGroup checks if a bound group matches the requested model.
// It first checks per-binding model_patterns, then falls back to the group's
// supported_model_scopes and platform-level heuristics.
func matchesGroup(bg *APIKeyGroup, requestedModel string) bool {
	// 1. Per-binding model_patterns take precedence
	if len(bg.ModelPatterns) > 0 {
		return matchesAnyPattern(bg.ModelPatterns, requestedModel)
	}

	// 2. Group-level supported_model_scopes
	if len(bg.Group.SupportedModelScopes) > 0 {
		return matchesModelScopes(bg.Group.SupportedModelScopes, requestedModel)
	}

	// 3. Platform-based heuristic fallback
	return matchesPlatformHeuristic(bg.Group.Platform, requestedModel)
}

// matchesAnyPattern checks if the model matches any of the given patterns.
// Supports exact match and trailing wildcard (e.g., "claude-*").
func matchesAnyPattern(patterns []string, model string) bool {
	for _, pattern := range patterns {
		if matchModelPatternMultiGroup(pattern, model) {
			return true
		}
	}
	return false
}

// matchModelPatternMultiGroup checks if a model matches a pattern.
// Supports exact match, trailing wildcard (*), and prefix match.
func matchModelPatternMultiGroup(pattern, model string) bool {
	if pattern == model {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(model, prefix)
	}
	return false
}

// matchesModelScopes checks if the model matches any supported model scope.
// Scopes like "claude", "gemini_text", "gpt" are treated as prefix matches.
func matchesModelScopes(scopes []string, model string) bool {
	modelLower := strings.ToLower(model)
	for _, scope := range scopes {
		scopeLower := strings.ToLower(scope)
		// Scope names are treated as prefixes (e.g., "claude" matches "claude-opus-4")
		if strings.HasPrefix(modelLower, scopeLower) {
			return true
		}
		// Also handle underscore-to-hyphen conversion (e.g., "gemini_text" → "gemini-text")
		scopeHyphen := strings.ReplaceAll(scopeLower, "_", "-")
		if strings.HasPrefix(modelLower, scopeHyphen) {
			return true
		}
	}
	return false
}

// matchesPlatformHeuristic uses platform name to guess model ownership.
// This is the lowest-priority fallback when no explicit patterns are configured.
func matchesPlatformHeuristic(platform, model string) bool {
	modelLower := strings.ToLower(model)
	switch strings.ToLower(platform) {
	case "anthropic":
		return strings.HasPrefix(modelLower, "claude")
	case "openai":
		return strings.HasPrefix(modelLower, "gpt") ||
			strings.HasPrefix(modelLower, "o1") ||
			strings.HasPrefix(modelLower, "o3") ||
			strings.HasPrefix(modelLower, "o4") ||
			strings.HasPrefix(modelLower, "chatgpt")
	case "gemini":
		return strings.HasPrefix(modelLower, "gemini")
	case "sora":
		return strings.HasPrefix(modelLower, "sora")
	default:
		return false
	}
}

// SortBoundGroupsByPriority sorts bound groups by priority (ascending).
// This is a safety measure — groups should already be sorted from the DB query.
func SortBoundGroupsByPriority(groups []APIKeyGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority < groups[j].Priority
	})
}
