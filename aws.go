package assumerole

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-ini/ini"
)

// IsAWSAccessDeniedError indicates whether an error is an AWS "access denied"
// error.
func IsAWSAccessDeniedError(err error) bool {
	awsErr, ok := err.(awserr.Error)
	return (ok && awsErr.Code() == "AccessDenied")
}

// AWSProvider is an interface to AWS.
type AWSProvider interface {
	AssumeRole(roleARN string, sessionName string) (*TemporaryCredentials, error)
	AssumeRoleWithMFA(roleARN string, sessionName string, mfaDeviceARN string, mfaToken string) (*TemporaryCredentials, error)
	MFADevices() ([]string, error)
	Username() (string, error)
	CurrentPrincipalARN() (string, error)
}

// AWSConfigProvider is an interface to the AWS configuration (usually
// kept in files in ~/.aws).
type AWSConfigProvider interface {
	GetCredentials(profileName string) (*TemporaryCredentials, error)
	SetCredentials(profileName string, creds *TemporaryCredentials) error
	GetProfile(profileName string) (*ProfileConfiguration, error)
	SetProfile(profileName string, profile *ProfileConfiguration) error
}

// ProfileConfiguration holds the configuration from a single profile
// usually in ~/.aws/config.
type ProfileConfiguration struct {
	Expires         time.Time
	MFASerial       string
	SourceProfile   string
	RoleARN         string
	RoleSessionName string
}

// TemporaryCredentials is a set of Amazon security credentials, along
// with an expiry.
type TemporaryCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expires         time.Time
}

// AWS is the default implementation of AWSProvider that talks to the
// real AWS.
type AWS struct {
	iam *iam.IAM
	sts *sts.STS
}

// NewAWS creates a new connection to AWS.
func NewAWS() (AWSProvider, error) {
	session, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}
	return &AWS{
		iam: iam.New(session),
		sts: sts.New(session),
	}, nil
}

// AssumeRole calls sts:AssumeRole and returns temporary credentials.
func (a *AWS) AssumeRole(roleARN string, sessionName string) (*TemporaryCredentials, error) {
	return a.AssumeRoleWithMFA(roleARN, sessionName, "", "")
}

// AssumeRoleWithMFA calls sts:AssumeRole (with MFA information) and
// returns temporary credentials.
func (a *AWS) AssumeRoleWithMFA(roleARN string, sessionName string, mfaDeviceARN string, mfaToken string) (*TemporaryCredentials, error) {
	req := &sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(int64(time.Hour.Seconds())),
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
	}

	if mfaDeviceARN != "" {
		req.SerialNumber = aws.String(mfaDeviceARN)
	}

	if mfaToken != "" {
		req.TokenCode = aws.String(mfaToken)
	}

	res, err := a.sts.AssumeRole(req)
	if err != nil {
		return nil, err
	}

	return &TemporaryCredentials{
		AccessKeyID:     *res.Credentials.AccessKeyId,
		Expires:         *res.Credentials.Expiration,
		SecretAccessKey: *res.Credentials.SecretAccessKey,
		SessionToken:    *res.Credentials.SessionToken,
	}, nil
}

// MFADevices lists the MFA devices on the current user's account.
func (a *AWS) MFADevices() ([]string, error) {
	username, err := a.Username()
	if err != nil {
		return nil, err
	}

	res, err := a.iam.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}

	devices := make([]string, len(res.MFADevices))
	for i := range res.MFADevices {
		devices[i] = *res.MFADevices[i].SerialNumber
	}
	return devices, nil
}

// Username returns the username of the current AWS user.
func (a *AWS) Username() (string, error) {
	res, err := a.iam.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return *res.User.UserName, nil
}

// CurrentPrincipalARN returns the ARN of the current IAM principal.
func (a *AWS) CurrentPrincipalARN() (string, error) {
	res, err := a.sts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *res.Arn, nil
}

// AWSConfig represents the default AWS config files that exist on a system at
// ~/.aws/{config,credentials}. These two files are inherently linked for us,
// because while the credentials are stored in the credentials file, the
// metadata about these credentials are stored in the profile config file.
type AWSConfig struct {
	config *AWSConfigOpts

	awsConfigIni      *ini.File
	awsCredentialsIni *ini.File
}

// AWSConfigOpts are the options for the AWSConfig.
type AWSConfigOpts struct {
	// ConfigFilePath is the path to the shared AWS config file, usually at
	// ~/.aws/config. If you leave this blank, the default location will be
	// used.
	ConfigFilePath string
	// CredentialsFilePath is the path to the shared AWS config file, usually
	// at ~/.aws/credentials. If you leave this blank, the default location
	// will be used.
	CredentialsFilePath string
}

// NewAWSConfig returns a new AWSConfig, that will lazily read credentials and
// configuration from the default AWS config at ~/.aws.
func NewAWSConfig(config AWSConfigOpts) (*AWSConfig, error) {
	if config.ConfigFilePath == "" {
		if os.Getenv("AWS_CONFIG_FILE") != "" {
			config.ConfigFilePath = os.Getenv("AWS_CONFIG_FILE")
		} else {
			config.ConfigFilePath = defaults.SharedConfigFilename()
		}
	}

	if config.CredentialsFilePath == "" {
		if os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != "" {
			config.CredentialsFilePath = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
		} else {
			config.CredentialsFilePath = defaults.SharedCredentialsFilename()
		}
	}

	awsConfigIni, err := ini.LooseLoad(config.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	awsCredentialsIni, err := ini.LooseLoad(config.CredentialsFilePath)
	if err != nil {
		return nil, err
	}

	return &AWSConfig{
		config:            &config,
		awsConfigIni:      awsConfigIni,
		awsCredentialsIni: awsCredentialsIni,
	}, nil
}

// credentialsIniSection returns the named INI section from the credentials
// file or creates it if it doesn't exist.
func (c *AWSConfig) credentialsIniSection(profileName string) (*ini.Section, error) {
	if section := c.awsCredentialsIni.Section(profileName); section != nil {
		return section, nil
	}

	return c.awsConfigIni.NewSection(profileName)
}

// credentialsIniSection returns the named INI section from the shared config
// file or creates it if it doesn't exist.
func (c *AWSConfig) profileIniSection(profileName string) (*ini.Section, error) {
	if section := c.awsConfigIni.Section(fmt.Sprintf("profile %s", profileName)); section != nil {
		return section, nil
	}

	return c.awsConfigIni.NewSection(profileName)
}

// GetProfile returns the AWS profile metadata information from the shared
// config file.
func (c *AWSConfig) GetProfile(profileName string) (*ProfileConfiguration, error) {
	section, err := c.profileIniSection(profileName)
	if err != nil {
		return nil, err
	}

	profileConfig := &ProfileConfiguration{}

	if key := section.Key("expiration"); key != nil {
		if value, err := key.TimeFormat(time.RFC3339); err == nil {
			profileConfig.Expires = value
		}
	}

	if key := section.Key("mfa_serial"); key != nil {
		profileConfig.MFASerial = key.String()
	}

	if key := section.Key("source_profile"); key != nil {
		profileConfig.SourceProfile = key.String()
	}

	if key := section.Key("role_arn"); key != nil {
		profileConfig.RoleARN = key.String()
	}

	if key := section.Key("role_session_name"); key != nil {
		profileConfig.RoleSessionName = key.String()
	}

	return profileConfig, nil
}

// SetProfile writes the specified profile information to the shared AWS config
// config file.
func (c *AWSConfig) SetProfile(profileName string, profile *ProfileConfiguration) error {
	section, err := c.profileIniSection(profileName)
	if err != nil {
		return err
	}

	if err := setIniKeyValue(section, "expiration", profile.Expires.Format(time.RFC3339)); err != nil {
		return err
	}

	if err := setIniKeyValue(section, "mfa_serial", profile.MFASerial); err != nil {
		return err
	}

	if err := setIniKeyValue(section, "source_profile", profile.SourceProfile); err != nil {
		return err
	}

	if err := setIniKeyValue(section, "role_arn", profile.RoleARN); err != nil {
		return err
	}

	if err := setIniKeyValue(section, "role_session_name", profile.RoleSessionName); err != nil {
		return err
	}

	// Ensure dir exists
	if err := os.MkdirAll(filepath.Dir(c.config.ConfigFilePath), 0755); err != nil {
		return err
	}

	return c.awsConfigIni.SaveTo(c.config.ConfigFilePath)
}

// GetCredentials retrieves the named credentials from the AWS credential file.
func (c *AWSConfig) GetCredentials(profileName string) (*TemporaryCredentials, error) {
	section, err := c.credentialsIniSection(profileName)
	if err != nil {
		return nil, err
	}

	creds := &TemporaryCredentials{}

	if key := section.Key("aws_access_key_id"); key != nil {
		creds.AccessKeyID = key.String()
	}

	if key := section.Key("aws_secret_access_key"); key != nil {
		creds.SecretAccessKey = key.String()
	}

	if key := section.Key("aws_session_token"); key != nil {
		creds.SessionToken = key.String()
	}

	// Get the expiry time from the profile
	profile, err := c.GetProfile(profileName)
	if err != nil {
		return nil, err
	}
	creds.Expires = profile.Expires

	return creds, nil
}

// SetCredentials saves the credentials to the AWS credential file.
func (c *AWSConfig) SetCredentials(profileName string, creds *TemporaryCredentials) error {
	section, err := c.credentialsIniSection(profileName)
	if err != nil {
		return err
	}

	if err := setIniKeyValue(section, "aws_access_key_id", creds.AccessKeyID); err != nil {
		return err
	}

	if err := setIniKeyValue(section, "aws_secret_access_key", creds.SecretAccessKey); err != nil {
		return err
	}

	if err := setIniKeyValue(section, "aws_session_token", creds.SessionToken); err != nil {
		return err
	}

	profile, err := c.profileIniSection(profileName)
	if err != nil {
		return err
	}

	// Set the expiry time in the profile
	if err := setIniKeyValue(profile, "expiration", creds.Expires.Format(time.RFC3339)); err != nil {
		return err
	}

	// Save profile
	if err := c.awsConfigIni.SaveTo(c.config.ConfigFilePath); err != nil {
		return err
	}

	// Ensure dir exists
	if err := os.MkdirAll(filepath.Dir(c.config.CredentialsFilePath), 0755); err != nil {
		return err
	}

	return c.awsCredentialsIni.SaveTo(c.config.CredentialsFilePath)
}
