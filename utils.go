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
