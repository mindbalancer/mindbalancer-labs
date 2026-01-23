// Package router provides request routing based on rules.
package router

import (
	"context"
	"regexp"
	"sort"
	"sync"

	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Router routes requests to hostgroups based on rules.
type Router struct {
	mu      sync.RWMutex
	storage *storage.Storage
	rules   []CompiledRule
}

// CompiledRule is a routing rule with compiled regex patterns.
type CompiledRule struct {
	Rule         storage.RoutingRule
	ModelRegex   *regexp.Regexp
	PatternRegex *regexp.Regexp
	UserRegex    *regexp.Regexp
}

// NewRouter creates a new router.
func NewRouter(store *storage.Storage) *Router {
	return &Router{
		storage: store,
		rules:   make([]CompiledRule, 0),
	}
}

// LoadRules loads routing rules from storage.
func (r *Router) LoadRules(ctx context.Context) error {
	rules, err := r.storage.GetRoutingRules(ctx)
	if err != nil {
		return err
	}

	compiled := make([]CompiledRule, 0, len(rules))
	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		cr := CompiledRule{Rule: rule}

		// Compile regex patterns
		if rule.MatchModel != "" {
			re, err := regexp.Compile(rule.MatchModel)
			if err == nil {
				cr.ModelRegex = re
			}
		}
		if rule.MatchPattern != "" {
			re, err := regexp.Compile(rule.MatchPattern)
			if err == nil {
				cr.PatternRegex = re
			}
		}
		if rule.MatchUser != "" {
			re, err := regexp.Compile(rule.MatchUser)
			if err == nil {
				cr.UserRegex = re
			}
		}

		compiled = append(compiled, cr)
	}

	// Sort by priority (lower = higher priority)
	sort.Slice(compiled, func(i, j int) bool {
		return compiled[i].Rule.Priority < compiled[j].Rule.Priority
	})

	r.mu.Lock()
	r.rules = compiled
	r.mu.Unlock()

	return nil
}

// RouteRequest determines the hostgroup for a request.
func (r *Router) RouteRequest(model, prompt, username string, defaultHostgroup int) (hostgroup int, mirror *int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rule := range r.rules {
		if r.matchRule(rule, model, prompt, username) {
			return rule.Rule.DestinationHostgroup, rule.Rule.MirrorHostgroup
		}
	}

	return defaultHostgroup, nil
}

func (r *Router) matchRule(rule CompiledRule, model, prompt, username string) bool {
	// Model match
	if rule.ModelRegex != nil {
		if !rule.ModelRegex.MatchString(model) {
			return false
		}
	} else if rule.Rule.MatchModel != "" {
		// Simple glob-like matching
		if !simpleMatch(rule.Rule.MatchModel, model) {
			return false
		}
	}

	// Pattern match (on prompt)
	if rule.PatternRegex != nil {
		if !rule.PatternRegex.MatchString(prompt) {
			return false
		}
	}

	// User match
	if rule.UserRegex != nil {
		if !rule.UserRegex.MatchString(username) {
			return false
		}
	} else if rule.Rule.MatchUser != "" {
		if rule.Rule.MatchUser != username {
			return false
		}
	}

	return true
}

// simpleMatch does simple glob-like matching with * wildcard.
func simpleMatch(pattern, s string) bool {
	if pattern == "" {
		return true
	}
	if pattern == "*" {
		return true
	}

	// Convert simple glob to regex
	regexPattern := "^"
	for _, c := range pattern {
		switch c {
		case '*':
			regexPattern += ".*"
		case '?':
			regexPattern += "."
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			regexPattern += "\\" + string(c)
		default:
			regexPattern += string(c)
		}
	}
	regexPattern += "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return pattern == s
	}
	return re.MatchString(s)
}

// GetRules returns all loaded rules.
func (r *Router) GetRules() []CompiledRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]CompiledRule, len(r.rules))
	copy(rules, r.rules)
	return rules
}

// AddRule adds a rule at runtime.
func (r *Router) AddRule(rule storage.RoutingRule) error {
	ctx := context.Background()
	if err := r.storage.InsertRoutingRule(ctx, &rule); err != nil {
		return err
	}
	return r.LoadRules(ctx)
}

// RemoveRule removes a rule at runtime.
func (r *Router) RemoveRule(ruleID int) error {
	ctx := context.Background()
	if err := r.storage.DeleteRoutingRule(ctx, ruleID); err != nil {
		return err
	}
	return r.LoadRules(ctx)
}
