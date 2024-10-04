package slackcmd

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type PatternError struct {
	Pattern string
}

func (e PatternError) Error() string {
	return fmt.Errorf("valid pattern is `%s`", e.Pattern).Error()
}

func patternError(pattern string) PatternError {
	return PatternError{Pattern: pattern}
}

var lockUnlockPattern = regexp.MustCompile(`(unlock|lock) ([0-9a-zA-Z-]+) (staging|production|sandbox|stg|pro|prd)\s*(.*)`)

func Parse(text string) (Command, error) {
	cmd, err1 := parseLockUnlock(text)
	if err1 == nil {
		return cmd, nil
	} else if !errors.As(err1, &PatternError{}) {
		return nil, fmt.Errorf("invalid command %q: %w", text, err1)
	}

	cmd, err2 := parseDescribeLocks(text)
	if err2 == nil {
		return cmd, nil
	} else if !errors.As(err2, &PatternError{}) {
		return nil, fmt.Errorf("invalid command %q: %w", text, err2)
	}

	return nil, fmt.Errorf("invalid command %q: %v, %v", text, err1, err2)
}

func parseLockUnlock(text string) (Command, error) {
	match := findLockUnlock(text)
	if match == nil {
		return nil, patternError("lock|unlock <project> <env> [for <reason>]")
	}

	var (
		command = match[0][1]
		project = match[0][2]
		env     = match[0][3]
		reason  = match[0][4]
	)

	switch command {
	case "unlock":
		if reason != "" {
			return nil, errors.New("unlock command does not accept reason")
		}

		return &Unlock{
			Project: project,
			Env:     env,
		}, nil
	case "lock":
		if reason == "" {
			return nil, errors.New("lock command requires reason")
		}

		if !strings.HasPrefix(reason, "for ") {
			return nil, errors.New("reason must start with 'for'")
		}

		reason = strings.TrimPrefix(reason, "for ")

		return &Lock{
			Project: project,
			Env:     env,
			Reason:  reason,
		}, nil
	default:
		panic("unreachable")
	}
}

func parseDescribeLocks(text string) (Command, error) {
	if !strings.Contains(text, "describe locks") {
		return nil, patternError("describe locks")
	}

	return &DescribeLocks{}, nil
}

func findLockUnlock(text string) [][]string {
	return lockUnlockPattern.FindAllStringSubmatch(text, -1)
}
