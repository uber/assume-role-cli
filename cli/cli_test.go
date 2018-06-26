package cli_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/uber/assume-role/cli"

	"github.com/hashicorp/go-multierror"
	"github.com/hgfischer/go-otp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Env vars containing secret keys need to be set for some integration tests
// to work properly. For security reasons, they are not committed to the repo.
var secretCredentialsEnvVarPrefix = "ASSUME_ROLE_INTEGRATION_TEST_"

// List of vars that make up the secret credentials.
var secretCredentialsVars = []string{
	"AWS_ACCESS_KEY",
	"AWS_SECRET_ACCESS_KEY",
}

var secretOTPEnvVar = secretCredentialsEnvVarPrefix + "AWS_OTP_SECRET"

type execTestOpts struct {
	// Args to send to the program for the test.
	args []string
	// Additional env vars to set before executing the test.
	env map[string]string
	// Directory that contains the fixture data (e.g. aws config files).
	// Test will be executed in this dir.
	fixture string
	// String value for stdin for the test.
	input string
}

type execResult struct {
	ExitCode int
	Stdout   *bytes.Buffer
	Stderr   *bytes.Buffer
}

func copyFilesToTempDir(src string) (string, error) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	// Copy the dir to a temporary location
	if err := exec.Command("cp", "-r", src, tmpdir).Run(); err != nil {
		return "", err
	}

	return filepath.Join(tmpdir, filepath.Base(src)), nil
}

func execTest(t *testing.T, opts execTestOpts) *execResult {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Restore existing env vars after changing them for the test
	oldAWSConfigFileEnv := os.Getenv("AWS_CONFIG_FILE")
	defer os.Setenv("AWS_CONFIG_FILE", oldAWSConfigFileEnv)
	oldAWSSharedCredsEnv := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	defer os.Setenv("AWS_SHARED_CREDENTIALS_FILE", oldAWSSharedCredsEnv)

	// Set additional env vars
	for key, val := range opts.env {
		oldValue := os.Getenv(key)
		defer os.Setenv(key, oldValue)

		os.Setenv(key, val)
	}

	os.Setenv("AWS_CONFIG_FILE", filepath.Join(opts.fixture, "aws/config"))
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(opts.fixture, "aws/credentials"))

	stdin := bytes.NewBufferString(opts.input)

	// Restore previous working dir after changing
	cwd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(opts.fixture)
	require.NoError(t, err)

	defer os.Chdir(cwd)

	exitCode := cli.Main(stdin, stdout, stderr, opts.args)

	return &execResult{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

// readSecretCredentials reads secret AWS credentials from environment
// variables. The AWS credentials are expected to be in an env var named like:
// "secretCredentialsEnvVarPrefix + credentialsKey + _AWS_XXXX".
func readSecretCredentialsFromEnv(credentialsKey string) (vars map[string]string, errs error) {
	vars = make(map[string]string)

	for _, targetEnvVarName := range secretCredentialsVars {
		sourceEnvVarName := fmt.Sprintf("%s%s_%s", secretCredentialsEnvVarPrefix, credentialsKey, targetEnvVarName)
		value := os.Getenv(sourceEnvVarName)

		if value == "" {
			errs = multierror.Append(errs, fmt.Errorf("missing required env var: %v", sourceEnvVarName))
		} else {
			vars[targetEnvVarName] = value
		}
	}

	if errs != nil {
		return nil, errs
	}

	return vars, nil
}

func TestAssumeRoleWithoutMFA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to -short flag")
	}

	awsCreds, err := readSecretCredentialsFromEnv("WITHOUT_MFA")
	require.NoError(t, err)

	fixtureDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(fixtureDir)

	result := execTest(t, execTestOpts{
		args:    []string{"--role", "arn:aws:iam::675470192105:role/test_assume-role"},
		env:     awsCreds,
		fixture: fixtureDir,
	})
	assert.Regexp(t, "^AWS_ACCESS_KEY_ID=.*\nAWS_SECRET_ACCESS_KEY=.*\nAWS_SESSION_TOKEN=.*\n$", result.Stdout.String())
	assert.Empty(t, result.Stderr.String())
	assert.Zero(t, result.ExitCode)
}

func TestErrorNoMFADevices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to -short flag")
	}

	awsCreds, err := readSecretCredentialsFromEnv("WITHOUT_MFA")
	require.NoError(t, err)

	fixtureDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(fixtureDir)

	result := execTest(t, execTestOpts{
		args:    []string{"--role", "arn:aws:iam::675470192105:role/no-access-role"},
		env:     awsCreds,
		fixture: fixtureDir,
	})
	assert.Contains(t, result.Stderr.String(), "error trying to AssumeRole without MFA")
	assert.Contains(t, result.Stderr.String(), "error trying to AssumeRole with MFA")
	assert.NotZero(t, result.ExitCode)
}

func TestAssumeRoleWithMFA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to -short flag")
	}

	awsCreds, err := readSecretCredentialsFromEnv("WITH_MFA")
	require.NoError(t, err)

	otpSecret := os.Getenv(secretOTPEnvVar)
	if otpSecret == "" {
		t.Errorf("missing OTP secret from env var: %v", secretOTPEnvVar)
		t.FailNow()
	}

	fixtureDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(fixtureDir)

	mfa := otp.TOTP{
		Secret:         otpSecret,
		IsBase32Secret: true,
	}

	result := execTest(t, execTestOpts{
		args:    []string{"--role", "arn:aws:iam::675470192105:role/test_assume-role"},
		env:     awsCreds,
		fixture: fixtureDir,
		input:   mfa.Get() + "\n",
	})
	assert.Regexp(t, "^AWS_ACCESS_KEY_ID=.*\nAWS_SECRET_ACCESS_KEY=.*\nAWS_SESSION_TOKEN=.*\n$", result.Stdout.String())
	assert.Equal(t, "Enter MFA token: ", result.Stderr.String())
	assert.Zero(t, result.ExitCode)
}

func TestCredentialsCached(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to -short flag")
	}

	awsCreds, err := readSecretCredentialsFromEnv("WITHOUT_MFA")
	require.NoError(t, err)

	fixtureDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(fixtureDir)

	// Do the first AssumeRole
	a := execTest(t, execTestOpts{
		args:    []string{"--role", "arn:aws:iam::675470192105:role/test_assume-role"},
		env:     awsCreds,
		fixture: fixtureDir,
	})
	require.Empty(t, a.Stderr.String())
	require.Zero(t, a.ExitCode)

	// Do the second AssumeRole
	b := execTest(t, execTestOpts{
		args:    []string{"--role", "arn:aws:iam::675470192105:role/test_assume-role"},
		env:     awsCreds,
		fixture: fixtureDir,
	})
	require.Empty(t, b.Stderr.String())
	require.Zero(t, b.ExitCode)

	// Credentials should match because they were cached the first time
	assert.Equal(t, a.Stdout.String(), b.Stdout.String())

	writtenCredentialFile, err := os.Stat(filepath.Join(fixtureDir, "aws/credentials"))
	require.NoError(t, err)

	writtenConfigFile, err := os.Stat(filepath.Join(fixtureDir, "aws/config"))
	require.NoError(t, err)

	// Config/credential files should have been written to
	assert.NotZero(t, writtenConfigFile.Size())
	assert.NotZero(t, writtenCredentialFile.Size())
}

func TestConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to -short flag")
	}

	awsCreds, err := readSecretCredentialsFromEnv("WITHOUT_MFA")
	require.NoError(t, err)

	fixtureDir, err := copyFilesToTempDir("fixtures/test-config")
	require.NoError(t, err)
	defer os.RemoveAll(fixtureDir)

	result := execTest(t, execTestOpts{
		args:    []string{"--role", "test_assume-role"},
		fixture: fixtureDir,
		env:     awsCreds,
	})
	assert.Empty(t, result.Stderr.String())
	assert.Zero(t, result.ExitCode)
}
