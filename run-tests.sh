#!/bin/bash
set -ev

# Encrypted env variables are not present in PRs generated
# from forks and therefore the tests will always fail.
# As the tests that need encrypted env variables are only
# integration tests, they can be skipped with short (-s) flag.
if [ "${TRAVIS_SECURE_ENV_VARS}" = "false" ]; then
    go test -v -short ./...
else
    go test -v ./...
fi
