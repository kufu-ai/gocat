package slackcmd

import (
	"fmt"
	"regexp"
	"strings"
)

var lockUnlockPattern = regexp.MustCompile(`(unlock|lock) ([0-9a-zA-Z-]+) (staging|production|sandbox|stg|pro|prd)\s*(.*)`)

func Parse(text string) (Command, error) {
	match := findLockUnlock(text)
	if match == nil {
		return nil, fmt.Errorf("invalid command %q: valid pattern is 'lock|unlock <project> <env> [for <reason>]", text)
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
			return nil, fmt.Errorf("invalid command %q: unlock command does not accept reason", text)
		}

		return &Unlock{
			Project: project,
			Env:     env,
		}, nil
	case "lock":
		if reason == "" {
			return nil, fmt.Errorf("invalid command %q: lock command requires reason", text)
		}

		if !strings.HasPrefix(reason, "for ") {
			return nil, fmt.Errorf("invalid command %q: reason must start with 'for'", text)
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

func findLockUnlock(text string) [][]string {
	return lockUnlockPattern.FindAllStringSubmatch(text, -1)
}
