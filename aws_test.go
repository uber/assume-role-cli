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
package assumerole_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/uber/assume-role-cli"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	awsConfig, err := assumerole.NewAWSConfig(assumerole.AWSConfigOpts{
		ConfigFilePath:      "fixtures/test-awsconfig/config",
		CredentialsFilePath: "fixtures/test-awsconfig/credentials",
	})
	require.NoError(t, err)

	fooTestProfile, err := awsConfig.GetProfile("foo-test")
	require.NoError(t, err)

	assert.Equal(t, &assumerole.ProfileConfiguration{
		Expires:         time.Date(2018, 4, 23, 13, 45, 43, 0, time.UTC),
		MFASerial:       "arn:aws:iam::123:mfa/bob",
		SourceProfile:   "default",
		RoleARN:         "arn:aws:iam::123:role/admin",
		RoleSessionName: "",
	}, fooTestProfile)
}

func TestWriteConfig(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	defer os.RemoveAll(tempDir)

	awsConfig, err := assumerole.NewAWSConfig(assumerole.AWSConfigOpts{
		ConfigFilePath:      filepath.Join(tempDir, "aws", "config"),
		CredentialsFilePath: filepath.Join(tempDir, "aws", "credentials"),
	})
	require.NoError(t, err)

	fooTestProfile, err := awsConfig.GetProfile("test")
	require.NoError(t, err)
	require.Nil(t, err)

	fooTestProfile = &assumerole.ProfileConfiguration{
		Expires:         time.Date(2018, 4, 23, 13, 45, 43, 0, time.UTC),
		MFASerial:       "arn:aws:iam::123:mfa/bob",
		SourceProfile:   "default",
		RoleARN:         "arn:aws:iam::123:role/admin",
		RoleSessionName: "",
	}

	err = awsConfig.SetProfile("test", fooTestProfile)
	require.NoError(t, err)

	fooTestProfileReRead, err := awsConfig.GetProfile("test")
	require.NoError(t, err)

	assert.Equal(t, fooTestProfile, fooTestProfileReRead)
}

func TestGetCredentialsFromAWSConfigFile(t *testing.T) {
	awsConfig, err := assumerole.NewAWSConfig(assumerole.AWSConfigOpts{
		ConfigFilePath:      "fixtures/test-getcredentials/config",
		CredentialsFilePath: "fixtures/test-getcredentials/credentials",
	})
	require.NoError(t, err)

	creds, err := awsConfig.GetCredentials("foo-test")
	require.NoError(t, err)

	assert.Equal(t, &assumerole.TemporaryCredentials{
		AccessKeyID:     "DEF",
		SecretAccessKey: "yyy",
		SessionToken:    "sss",
		Expires:         time.Date(2018, 4, 23, 13, 45, 43, 0, time.UTC),
	}, creds)
}

func TestWriteCredentialsToAWSConfigFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	defer os.RemoveAll(tempDir)

	awsConfig, err := assumerole.NewAWSConfig(assumerole.AWSConfigOpts{
		ConfigFilePath:      filepath.Join(tempDir, "config"),
		CredentialsFilePath: filepath.Join(tempDir, "credentials"),
	})
	require.NoError(t, err)

	fooCreds := &assumerole.TemporaryCredentials{
		AccessKeyID:     "DEF",
		SecretAccessKey: "yyy",
		SessionToken:    "sss",
		Expires:         time.Date(2018, 4, 23, 13, 45, 43, 0, time.UTC),
	}

	err = awsConfig.SetCredentials("foo-test", fooCreds)
	require.NoError(t, err)

	fooCredsReRead, err := awsConfig.GetCredentials("foo-test")
	require.NoError(t, err)

	assert.Equal(t, fooCreds, fooCredsReRead)
}
