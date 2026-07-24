package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPhaseAllowed(t *testing.T) {
	assert.True(t, isPhaseAllowed(nil, "production"))
	assert.True(t, isPhaseAllowed([]string{"staging"}, "stg"))
	assert.False(t, isPhaseAllowed([]string{"staging"}, "production"))
}
