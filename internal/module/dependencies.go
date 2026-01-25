package module

import (
	"regexp"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

var constraintPattern = regexp.MustCompile(`^(==|=|!=|>=|<=|>|<)?\s*([0-9]+(?:\.[0-9]+){0,2})$`)

func ParseModuleDependency(value string) (string, *VersionConstraint, error) {
	parts := strings.SplitN(value, "@", 2)
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", nil, i18n.Errorf("module name is required")
	}
	if len(parts) == 1 {
		return name, nil, nil
	}
	constraint, err := ParseConstraint(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", nil, err
	}
	return name, &constraint, nil
}

func ParseConstraint(value string) (VersionConstraint, error) {
	if value == "" {
		return VersionConstraint{}, i18n.Errorf("constraint is required")
	}
	matches := constraintPattern.FindStringSubmatch(value)
	if matches == nil {
		return VersionConstraint{}, i18n.Errorf("invalid constraint %q", value)
	}
	operator := matches[1]
	versionValue := matches[2]
	parsed, err := ParseVersion(versionValue)
	if err != nil {
		return VersionConstraint{}, err
	}
	if operator == "" {
		operator = "="
	}
	return VersionConstraint{Operator: operator, Version: parsed}, nil
}
