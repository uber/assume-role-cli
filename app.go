package assumerole

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/crypto/ssh/terminal"
)

// App is the main AssumeRole app.
type App struct {
	aws       AWSProvider
	awsConfig AWSConfigProvider
	clock     Clock
	config    Config
	stderr    io.Writer
	stdin     io.Reader

	stdinReader *bufio.Reader
}

// AssumeRoleParameters are the parameters for the AssumeRole call
type AssumeRoleParameters struct {
	// UserRole is the ARN of the role to be assumed
	UserRole string

	// RoleSessionName is the session name for the AWS AssumeRole call; if it is
	// the empty string, the current username will be used
	RoleSessionName string
}

// used here and in tests
var errAssumedRoleNeedsSessionName = errors.New("Validation error: missing role session name when current IAM principal is an assumed role")

// NewApp creates a new App.
func NewApp(opts ...Option) (*App, error) {
	app := &App{
		stdin:  os.Stdin,
		stderr: os.Stderr,
	}

	if err := app.applyOptions(opts...); err != nil {
		return nil, err
	}

	if err := app.setDefaults(); err != nil {
		return nil, err
	}

	app.stdinReader = bufio.NewReader(app.stdin)

	return app, nil
}

// AssumeRole takes a role name and calls AWS AssumeRole, returning a
// set of temporary credentials. If MFA is required, it will prompt for
// an MFA token interactively.
func (app *App) AssumeRole(options AssumeRoleParameters) (*TemporaryCredentials, error) {
	profileName, err := app.profileName(options.UserRole)
	if err != nil {
		return nil, err
	}

	profile, err := app.awsConfig.GetProfile(profileName)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		profile = &ProfileConfiguration{}
	}

	currentPrincipalIsAssumedRole, err := app.CurrentPrincipalIsAssumedRole()
	if err != nil {
		return nil, fmt.Errorf("ERROR while checking IAM principal type: %v", err)
	}

	// If the credentials from a previous session are still valid,
	// return those
	if !app.credentialsExpired(profile.Expires) {
		return app.awsConfig.GetCredentials(profileName)
	}

	// Get the full role ARN by combining the role prefix with the
	// user-provided role name
	roleARN := fmt.Sprintf("%s%s", app.config.RolePrefix, options.UserRole)
	profile.RoleARN = roleARN

	sessionName := profile.RoleSessionName
	if sessionName == "" {
		if options.RoleSessionName != "" {
			sessionName = options.RoleSessionName
		} else {
			if currentPrincipalIsAssumedRole {
				return nil, errAssumedRoleNeedsSessionName
			}
			sessionName, err = app.aws.Username()
			if err != nil {
				return nil, fmt.Errorf("unable to get username from AWS: %v", err)
			}
		}
		profile.RoleSessionName = sessionName
	}

	// We first try to assume role without MFA and if that doesn't work then we
	// try to assume role with MFA. Along the way, we collect errors in a
	// multierr, so that if there is a fatal problem then we can output all
	// errors so the user can see what happened along the way.
	var finalErr error

	// Try to assume role without MFA
	creds, err := app.aws.AssumeRole(roleARN, sessionName)
	if err != nil {
		if IsAWSAccessDeniedError(err) {
			finalErr = multierror.Append(finalErr, fmt.Errorf("error trying to AssumeRole without MFA: %v", err))
		} else {
			// Fail immediately if the error was something other than "access denied"
			return nil, err
		}
	}
	if creds != nil {
		profile.Expires = creds.Expires

		// Save credentials
		if err := app.save(profileName, profile, creds); err != nil {
			return nil, err
		}

		return creds, nil
	}

	if currentPrincipalIsAssumedRole {
		// assumed roles don't have an user name or MFA device associated with them
		return nil, finalErr
	}

	// Get user's MFA device
	mfaDeviceARN, err := app.mfaDevice()
	if err != nil {
		finalErr = multierror.Append(finalErr, fmt.Errorf("error trying to AssumeRole with MFA: %v", err))
		return nil, finalErr
	}
	profile.MFASerial = mfaDeviceARN

	// Get token
	mfaToken, err := app.mfaToken()
	if err != nil {
		finalErr = multierror.Append(finalErr, fmt.Errorf("error trying to AssumeRole with MFA: %v", err))
		return nil, finalErr
	}

	// Assume role
	creds, err = app.aws.AssumeRoleWithMFA(roleARN, sessionName, mfaDeviceARN, mfaToken)
	if err != nil {
		finalErr = multierror.Append(finalErr, fmt.Errorf("error trying to AssumeRole with MFA: %v; giving up", err))
		return nil, finalErr
	}
	profile.Expires = creds.Expires

	// Save credentials
	if err := app.save(profileName, profile, creds); err != nil {
		return nil, err
	}

	return creds, nil
}

// CurrentPrincipalIsAssumedRole returns true is the current principal is an assumed role.
func (app *App) CurrentPrincipalIsAssumedRole() (bool, error) {
	arn, err := app.aws.CurrentPrincipalARN()
	if err != nil {
		return false, err
	}
	return regexp.MatchString(`^arn:aws:sts::[0-9]+:assumed-role/`, arn)
}

// credentialsExpired returns a boolean indicating whether the credentials
// are still valid. This is based on the credentials expiry and the refresh
// horizon configuration.
func (app *App) credentialsExpired(expiryTime time.Time) bool {
	return app.clock.Now().After(expiryTime.Add(-app.config.RefreshBeforeExpiry))
}

func (app *App) mfaDevice() (string, error) {
	devices, err := app.aws.MFADevices()
	if err != nil {
		return "", err
	}
	if len(devices) < 1 {
		return "", errors.New("no MFA devices found")
	}
	if len(devices) == 1 {
		return devices[0], nil
	}

Prompt:
	for i, device := range devices {
		fmt.Fprintf(app.stderr, "[%d]: %s\n", i+1, device)
	}

	app.stderr.Write([]byte("Select MFA device: "))

	userInput, err := readInput(app.stdinReader)
	if err != nil {
		return "", fmt.Errorf("unable to read MFA device option from stdin: %v", err)
	}

	userInputInt, err := strconv.Atoi(userInput)
	if err != nil {
		app.stderr.Write([]byte("Invalid input (not a number)\n"))
		goto Prompt
	}

	if userInputInt < 1 || userInputInt > len(devices) {
		app.stderr.Write([]byte("Invalid input (not in range)\n"))
		goto Prompt
	}

	return devices[userInputInt-1], nil
}

func (app *App) mfaToken() (string, error) {
	var token string
	var err error

	app.stderr.Write([]byte("Enter MFA token: "))

	stdinFile, ok := app.stdin.(*os.File)

	if ok && terminal.IsTerminal(int(stdinFile.Fd())) {
		token, err = readSecretInputFromTerminal(stdinFile)
		// Echo the user's "enter" keypress so they get feedback that they did
		// in fact hit enter.
		app.stderr.Write([]byte("\n"))
	} else {
		token, err = readInput(app.stdinReader)
	}

	if err != nil {
		return "", fmt.Errorf("unable to read MFA token from stdin: %v", err)
	}

	return strings.TrimSpace(token), nil
}

// profileName returns a string that will be used as the profile name
// in the AWS config for these credentials.
func (app *App) profileName(userRole string) (string, error) {
	var profileNamePrefix string

	roleARN, err := app.roleARN(userRole)
	if err != nil {
		return "", err
	}

	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return "", err
	}

	if app.config.ProfileNamePrefix != "" {
		profileNamePrefix = app.config.ProfileNamePrefix
	} else {
		profileNamePrefix = parsedARN.AccountID
	}

	return fmt.Sprintf("%s-%s", profileNamePrefix, filepath.Base(parsedARN.Resource)), nil
}

// roleARN returns the full role ARN, based on configuration and what
// is provided.
func (app *App) roleARN(userRole string) (string, error) {
	if isValidARN(userRole) {
		return userRole, nil
	}

	// Combine the user provided role name with the prefix from the
	// config.
	combined := fmt.Sprintf("%s%s", app.config.RolePrefix, userRole)

	if isValidARN(combined) {
		return combined, nil
	}

	return "", fmt.Errorf("invalid role ARN: %v", combined)
}

// save the credentials and profile.
func (app *App) save(profileName string, profile *ProfileConfiguration, creds *TemporaryCredentials) error {
	if err := app.awsConfig.SetProfile(profileName, profile); err != nil {
		return err
	}
	if err := app.awsConfig.SetCredentials(profileName, creds); err != nil {
		return err
	}

	return nil
}

func (app *App) setDefaults() error {
	if app.aws == nil {
		defaultAWS, err := NewAWS()
		if err != nil {
			return err
		}
		app.aws = defaultAWS
	}

	if app.awsConfig == nil {
		defaultCfg, err := NewAWSConfig(AWSConfigOpts{})
		if err != nil {
			return err
		}
		app.awsConfig = defaultCfg
	}

	if app.clock == nil {
		app.clock = &defaultClock{}
	}

	app.config.setDefaults()

	return nil
}
