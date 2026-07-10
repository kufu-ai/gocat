package main

import (
	"fmt"
	"strings"

	"github.com/zaiminc/gocat/slackcmd"
)

func parseAllowedPhases(value string) []string {
	var phases []string
	for _, phase := range strings.Split(value, ",") {
		phase = strings.TrimSpace(phase)
		if phase == "" {
			continue
		}
		phases = append(phases, slackcmd.NormalizeEnv(phase))
	}
	return phases
}

func isPhaseAllowed(allowedPhases []string, phase string) bool {
	if len(allowedPhases) == 0 {
		return true
	}

	phase = slackcmd.NormalizeEnv(phase)
	for _, allowedPhase := range allowedPhases {
		if slackcmd.NormalizeEnv(allowedPhase) == phase {
			return true
		}
	}
	return false
}

func disallowedPhaseError(phase string) error {
	return fmt.Errorf("phase %s is not allowed", slackcmd.NormalizeEnv(phase))
}
