package module

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

var versionPattern = regexp.MustCompile(`^\d+(?:\.\d+){0,2}$`)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(value string) (Version, error) {
	if !versionPattern.MatchString(value) {
		return Version{}, i18n.Errorf("invalid version format: %q", value)
	}

	parts := strings.Split(value, ".")
	parsed := []int{0, 0, 0}
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, i18n.Errorf("invalid version segment %q: %w", part, err)
		}
		parsed[i] = number
	}

	return Version{Major: parsed[0], Minor: parsed[1], Patch: parsed[2]}, nil
}

func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	return compareInt(v.Patch, other.Patch)
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

type VersionConstraint struct {
	Operator string
	Version  Version
}

func (constraint VersionConstraint) Match(version Version) bool {
	comparison := version.Compare(constraint.Version)
	switch constraint.Operator {
	case "", "=", "==":
		return comparison == 0
	case "!=":
		return comparison != 0
	case ">":
		return comparison > 0
	case ">=":
		return comparison >= 0
	case "<":
		return comparison < 0
	case "<=":
		return comparison <= 0
	default:
		return false
	}
}
