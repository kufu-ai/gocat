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

const (
	projectPattern = `([0-9a-zA-Z-]+)`
	envPattern     = `(staging|production|sandbox|stg|pro|prd)`
)

var (
	mentionPattern               = regexp.MustCompile(`^<@[A-Z0-9]+>$`)
	lockUnlockPattern            = regexp.MustCompile(`^(unlock|lock) ` + projectPattern + ` ` + envPattern + `\s*(.*)$`)
	deployBranchListPattern      = regexp.MustCompile(`^deploy ` + projectPattern + ` ` + envPattern + ` branch$`)
	deployPattern                = regexp.MustCompile(`^deploy ` + projectPattern + ` ` + envPattern + `$`)
	deployTargetSelectionPattern = regexp.MustCompile(`^deploy ` + envPattern + `$`)
)

func Parse(text string) (Command, error) {
	commandText := trimMention(text)

	cmd, err0 := parseBasic(commandText)
	if err0 == nil {
		return cmd, nil
	} else if !errors.As(err0, &PatternError{}) {
		return nil, fmt.Errorf("invalid command %q: %w", text, err0)
	}

	cmd, err1 := parseLockUnlock(commandText)
	if err1 == nil {
		return cmd, nil
	} else if !errors.As(err1, &PatternError{}) {
		return nil, fmt.Errorf("invalid command %q: %w", text, err1)
	}

	cmd, err2 := parseDescribeLocks(commandText)
	if err2 == nil {
		return cmd, nil
	} else if !errors.As(err2, &PatternError{}) {
		return nil, fmt.Errorf("invalid command %q: %w", text, err2)
	}

	cmd, err3 := parseDeploy(commandText)
	if err3 == nil {
		return cmd, nil
	} else if !errors.As(err3, &PatternError{}) {
		return nil, fmt.Errorf("invalid command %q: %w", text, err3)
	}

	return nil, fmt.Errorf("invalid command %q: %v, %v, %v, %v", text, err0, err1, err2, err3)
}

func trimMention(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	if mentionPattern.MatchString(fields[0]) {
		fields = fields[1:]
	}

	return strings.Join(fields, " ")
}

func parseBasic(text string) (Command, error) {
	switch text {
	case "help":
		return &Help{}, nil
	case "ls":
		return &ListProjects{}, nil
	case "reload":
		return &Reload{}, nil
	default:
		return nil, patternError("help|ls|reload")
	}
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
	env = NormalizeEnv(env)

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
	if text != "describe locks" {
		return nil, patternError("describe locks")
	}

	return &DescribeLocks{}, nil
}

func parseDeploy(text string) (Command, error) {
	if match := deployBranchListPattern.FindStringSubmatch(text); match != nil {
		return &DeployBranchList{
			Project: match[1],
			Env:     NormalizeEnv(match[2]),
		}, nil
	}

	if match := deployPattern.FindStringSubmatch(text); match != nil {
		return &Deploy{
			Project: match[1],
			Env:     NormalizeEnv(match[2]),
		}, nil
	}

	if match := deployTargetSelectionPattern.FindStringSubmatch(text); match != nil {
		return &DeployTargetSelection{
			Env: NormalizeEnv(match[1]),
		}, nil
	}

	return nil, patternError("deploy <project> <env> [branch]|deploy <env>")
}

func findLockUnlock(text string) [][]string {
	return lockUnlockPattern.FindAllStringSubmatch(text, -1)
}

func NormalizeEnv(env string) string {
	switch env {
	case "pro", "prd", "production":
		return "production"
	case "stg", "staging":
		return "staging"
	default:
		return env
	}
}
