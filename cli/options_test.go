package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testRole = "arn:aws:iam::675470192105:role/test_assume-role"

func TestParseOptionsCommonCase(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "ls", "-l"}, false)
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, "", cliOpts.roleSessionName)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsSessionName(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "--role-session-name", "test-session-name", "ls", "-l"}, false)
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, "test-session-name", cliOpts.roleSessionName)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsDoubleDash(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "--", "ls", "-l"}, false)
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, "", cliOpts.roleSessionName)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsNoRole(t *testing.T) {
	_, err := parseOptions([]string{"ls", "-l"}, false)
	assert.Error(t, err)
	assert.Equal(t, errNoRole, err)
}

func TestParseOptionsNoRoleSessionName(t *testing.T) {
	_, err := parseOptions([]string{"--role", testRole, "ls", "-l"}, true)
	assert.Error(t, err)
	assert.Equal(t, errAssumedRoleNeedsSessionName, err)
}
