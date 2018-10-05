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
package assumerole

import (
	"bufio"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/go-ini/ini"
	"golang.org/x/crypto/ssh/terminal"
)

func isValidARN(str string) bool {
	_, err := arn.Parse(str)
	return err == nil
}

func readInput(in *bufio.Reader) (string, error) {
	val, err := in.ReadString('\n')
	return strings.TrimSpace(val), err
}

func readSecretInputFromTerminal(in *os.File) (string, error) {
	b, err := terminal.ReadPassword(int(in.Fd()))
	return string(b), err
}

func setIniKeyValue(section *ini.Section, key string, value string) error {
	section.DeleteKey(key)
	_, err := section.NewKey(key, value)
	return err
}
