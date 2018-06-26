# assume-role

assume-role is a CLI tool for running programs with temporary AWS credentials. It is intended to be used by operators for running scripts and other tools that don't have native AssumeRole support.

**Example**

Run `myscript.py` using the "admin" role in your AWS account:

```
assume-role --role admin ./myscript.py
```

## Features

* Caches credentials with configurable expiry time (e.g. 15 mins before credentials are due to expire)
* Interoperability with awscli
* Supports MFA and attempts to autodetect when MFA is required
* Configurable via autoloading config file

## Getting started

**Set up base policy**

assume-role requires the user performing the AssumeRole call has the `iam:GetUser` permission, to identify the username and use that as the session name (so the user's name shows up in the CloudTrail UI).

If MFA needs to be provided, assume-role also requires that the current user can list their own MFA devices.

Create the following policy (e.g. named "allow-assume-role-script") and attach this to users or groups who will be performing the AssumeRole:

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "iam:GetUser",
                "iam:ListMFADevices"
            ],
            "Resource": "arn:aws:iam::<account-ID>:user/${aws:username}"
        }
    ]
}
```

(Replace `<account-ID>` with your AWS account ID.)

**Create a configuration file**

The `--role` option takes the full ARN of the role you want to assume (e.g. `arn:aws:iam::1234567890:role/admin`). To save humans on typing, you can specify the role prefix in configuration, so that you only need to use the named part of the role (i.e. `--role admin`).

Create a YAML-based configuration file called `assume-role.yaml` in root of your project directory:

```
role_prefix: arn:aws:iam::1234567890:role/
```

This allows you to execute assume-role by using the short role name, e.g.:

```
assume-role --role myrolename
```

To see other useful configuration, see *Configuration options* below.

**Install and run it**

You can install assume-role using go get:

```
go get -u github.com/uber/assume-role
```

Now, run a command:

```
assume-role --role admin ./myscript.py
```

If you run it without a command to execute, environment variables will be printed to the console instead:

```
assume-role --role admin
AWS_ACCESS_KEY_ID=ASIAQWERTYUIOPASDFGHJKL
AWS_SECRET_ACCESS_KEY=8qLCbGYKhOWXU38ZVj+RhY1f7+zvuZ3vHMIhNGTxnhs=
AWS_SESSION_TOKEN=Wt5owtYQ/zObHy+8KLAgejM/CKGlt3Fa67PpRt+dVaDv4+NqmuFBu6VCkV1jmtfr82eABf9R2sN76ezZ1NIaaKnnkx8fk1WIH7jb7e5KYD0gsaOaAFIKEsMBMixvrFcxTe4Xth8D7lCohZZLTU2I2kazJxOrE249Xwq61hh1ZTezKHNvqek9BbItQdaWoniEkJz9vtTgXYSxnBJoV+VIsSa7KyDcLrteHVKdLx7qkxvsZvXkvmPRnQtnrGBeT3pm7LIlc2xOiKgAxuDf8gW5RWORrz71DdzFfPVqi0lAw5Hx0Qx/9gipuTPr5DICUzah8l64w4t21R0L9T1r84NAjA==
```

That's it!

## Configuration options

Configuration is done by placing a file named `assume-role.yaml` in your project directory, or in `~/.aws`.

assume-role will locate this file if you are running it from within a project subdirectory.

The following configuration options are available:

* `refresh_before_expiry: <duration>` (default `15m`)

    When you run assume-role credentials are cached and subsequent invocations just read from the cache. When the credentials expire, a refresh is triggered (doing the AssumeRole again).
    
    This value controls how long before the credentials are due to expire we'll refresh them anyway. This is so that credentials don't expire in the middle of running a command.

* `role_prefix: <string>` (default: empty)

    To avoid typing the full ARN at the command-line every time, you can a prefix so you no longer have to type:
    
    ```
    assume-role --role arn:aws:iam::123:role/admin
    ```
    
    Instead you can do:
    
    ```
    assume-role --role admin
    ```
    
    By configuring `arn:aws:iam::123:role/` as the prefix.

* `profile_name_prefix: <string>` (defaults to empty, which uses your AWS account ID instead)

    When you do an assume-role, the credentials are saved to `~/.aws/credentials` under a name in the format `<profile_name_prefix>-<role_name>`. This allows you to then use the profile with other tools using the `AWS_PROFILE` variable, or for example when executing awscli directly: `aws --profile=myaccount-admin s3 ls bucket://mybucket/`.

    This is a convenience helper but is generally not needed if you always just run all your commands through assume-role.
