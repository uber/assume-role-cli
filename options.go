package assumerole

import (
	"io"

	multierror "github.com/hashicorp/go-multierror"
)

// Option is an option for the App that allows for changing of options or
// dependency injection for testing.
type Option func(*App) error

func (app *App) applyOptions(opts ...Option) (errs error) {
	for _, opt := range opts {
		if err := opt(app); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// WithAWS allows you to pass a custom AWSProvider for talking to AWS.
func WithAWS(aws AWSProvider) Option {
	return func(app *App) error {
		app.aws = aws
		return nil
	}
}

// WithAWSConfig allows you to pass a custom AWSConfigProvider, which stores
// config and credentials for talking to AWS.
func WithAWSConfig(awsConfig AWSConfigProvider) Option {
	return func(app *App) error {
		app.awsConfig = awsConfig
		return nil
	}
}

// WithClock allows you to specify a custom clock implementation (for tests).
func WithClock(clock Clock) Option {
	return func(app *App) error {
		app.clock = clock
		return nil
	}
}

// WithConfig allows you to customise the configuration for the AssumeRole app
// itself.
func WithConfig(config *Config) Option {
	return func(app *App) error {
		app.config = *config
		return nil
	}
}

// WithStderr allows you to pass a custom stderr.
func WithStderr(stderr io.Writer) Option {
	return func(app *App) error {
		app.stderr = stderr
		return nil
	}
}

// WithStdin allows you to pass a custom stdin.
func WithStdin(stdin io.Reader) Option {
	return func(app *App) error {
		app.stdin = stdin
		return nil
	}
}
