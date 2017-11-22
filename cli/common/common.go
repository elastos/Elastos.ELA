package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"DNA_POW/common/config"
	"DNA_POW/common/password"

	"github.com/urfave/cli"
)

func Address() string {
	return "http://localhost" + ":" + strconv.Itoa(config.Parameters.HttpJsonPort)
}

func PrintError(c *cli.Context, err error, cmd string) {
	fmt.Println("Incorrect Usage:", err)
	fmt.Println("")
	cli.ShowCommandHelp(c, cmd)
}

func FormatOutput(o []byte) error {
	var out bytes.Buffer
	err := json.Indent(&out, o, "", "\t")
	if err != nil {
		return err
	}
	out.Write([]byte("\n"))
	_, err = out.WriteTo(os.Stdout)

	return err
}

// WalletPassword prompts user to input wallet password when password is not
// specified from command line
func WalletPassword(passwd string) []byte {
	if passwd == "" {
		tmppasswd, _ := password.GetPassword()
		return tmppasswd
	} else {
		return []byte(passwd)
	}
}
