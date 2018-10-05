/*
 *  Copyright (c) 2018 Uber Technologies, Inc.
 *
 *     Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
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
	assert.Equal(t, "", cliOpts.roleSessionName)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsSessionName(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "--role-session-name", "test-session-name", "ls", "-l"})
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, "test-session-name", cliOpts.roleSessionName)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsDoubleDash(t *testing.T) {
	cliOpts, err := parseOptions([]string{"--role", testRole, "--", "ls", "-l"})
	assert.NoError(t, err)
	assert.Equal(t, testRole, cliOpts.role)
	assert.Equal(t, "", cliOpts.roleSessionName)
	assert.Equal(t, []string{"ls", "-l"}, cliOpts.args)
}

func TestParseOptionsNoRole(t *testing.T) {
	_, err := parseOptions([]string{"ls", "-l"})
	assert.Error(t, err)
	assert.Equal(t, errNoRole, err)
}
