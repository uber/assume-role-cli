package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testRole = "arn:aws:iam::675470192105:role/test_assume-role"

func TestParseOptionsCommonCase(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "ls", "-l"})
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsDoubleDash(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "--", "ls", "-l"})
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsNoRole(t *testing.T) {
	_, err := parseOptions([]string{"ls", "-l"})
	assert.Error(t, err)
	assert.Equal(t, errNoRole, err)
}
