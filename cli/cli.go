package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/uber/assume-role"
)

// credentialsToEnv takes credentials and outputs them as a list of environment
// variables.
func credentialsToEnv(creds *assumerole.TemporaryCredentials) (envVars []string) {
	envVars = append(envVars,
		fmt.Sprintf("%s=%s", "AWS_ACCESS_KEY_ID", creds.AccessKeyID),
		fmt.Sprintf("%s=%s", "AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey),
		fmt.Sprintf("%s=%s", "AWS_SESSION_TOKEN", creds.SessionToken),
	)

	return envVars
}

// execute will act like "exec <cmd> [args ...]" in a shell, first searching
// for cmd in the path and then ecxecuting it, replacing the current running
// process.
func execute(cmd string, args []string, env []string) error {
	binary, err := exec.LookPath(cmd)
	if err != nil {
		return err
	}

	// execve will replace the current running process on success
	return syscall.Exec(binary, args, env)
}

func loadApp(stdin io.Reader, stdout io.Writer, stderr io.Writer) (*assumerole.App, error) {
	appOpts := []assumerole.Option{
		assumerole.WithStdin(stdin),
		assumerole.WithStderr(stderr),
	}

	configFile, err := findConfigFile()
	if err != nil {
		return nil, err
	}

	if configFile != "" {
		config, err := assumerole.LoadConfig(configFile)
		if err != nil {
			return nil, err
		}

		appOpts = append(appOpts, assumerole.WithConfig(config))
	}

	return assumerole.NewApp(appOpts...)
}

func printHelp(out io.Writer) {
	fmt.Fprint(out, `Assume an AWS role and run the specified command.

Usage:
  assume-role [options] <command> [args ...]

Options:
      --help                       Help for assume-role
      --role string                Name of the role to assume
      --role-session-name string   Name of the session for the assumed role
`)
}

func printVars(vars []string, out io.Writer) {
	for _, x := range vars {
		fmt.Fprintf(out, "%s\n", x)
	}
}

// Main is the main entry point into the CLI program.
func Main(stdin io.Reader, stdout io.Writer, stderr io.Writer, args []string) (exitCode int) {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printHelp(stdout)
		return 0
	}

	app, err := loadApp(stdin, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	currentPrincipalIsAssumedRole, err := app.CurrentPrincipalIsAssumedRole()
	if err != nil {
		fmt.Printf("ERROR while checking IAM principal type: %v", err)
		return 1
	}

	userOpts, err := parseOptions(args, currentPrincipalIsAssumedRole)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	credentials, err := app.AssumeRole(assumerole.AssumeRoleParameters{
		UserRole:        userOpts.role,
		RoleSessionName: userOpts.roleSessionName,
	})
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	vars := credentialsToEnv(credentials)

	if len(userOpts.args) == 0 {
		// Print vars to stdout
		printVars(vars, stdout)
	} else {
		// Add AWS credentials to the environment
		env := append(os.Environ(), vars...)

		// execve will replace the current running process on success
		if err := execute(userOpts.args[0], userOpts.args, env); err != nil {
			fmt.Fprintf(stderr, "ERROR: Could not execute command: %v\n", err)
			return 127
		}
	}

	return 0
}
