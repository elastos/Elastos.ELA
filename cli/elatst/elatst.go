package elatst

import (
	. "ELA/cli/common"
	"ELA/cli/elatst/elaapi"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	//. "ELA/common"

	"github.com/urfave/cli"
	"github.com/yuin/gopher-lua"
)

func WalkDir(dirPth, suffix string) (files []string, err error) {
	files = make([]string, 0, 30)
	suffix = strings.ToUpper(suffix)
	err = filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error {
		//if err != nil {
		// return err
		//}
		if fi.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, filename)
		}
		return nil
	})
	return files, err
}

func elaTstAction(c *cli.Context) (err error) {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	fileTest := c.Bool("file")
	dirTest := c.Bool("dir")
	strTest := c.Bool("str")
	content := c.String("content")

	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("elaapi", elaapi.Loader)
	elaapi.RegisterDataType(L)
	//elaapi.RegisterAssetType(L)
	//elaapi.RegisterBalanceTxInputType(L)
	//elaapi.RegisterClientType(L)
	//elaapi.RegisterFunctionCodeType(L)
	//elaapi.RegisterTxAttributeType(L)
	//elaapi.RegisterUTXOTxInputType(L)
	//elaapi.RegisterTxOutputType(L)
	//elaapi.RegisterBookKeeperType(L)
	//elaapi.RegisterBookKeepingType(L)
	//elaapi.RegisterCoinBaseType(L)
	//elaapi.RegisterIssueAssetType(L)
	//elaapi.RegisterTransferAssetType(L)
	//elaapi.RegisterRegisterAssetType(L)
	//elaapi.RegisterRecordType(L)
	//elaapi.RegisterDataFileType(L)
	//elaapi.RegisterPrivacyPayloadType(L)
	//elaapi.RegisterDeployCodeType(L)
	//elaapi.RegisterTransactionType(L)
	//elaapi.RegisterBlockdataType(L)
	//elaapi.RegisterBlockType(L)

	if strTest {
		fmt.Println("str test")
		if err := L.DoString(content); err != nil {
			panic(err)
		}
	}

	if fileTest {
		fmt.Println("file test")
		if err := L.DoFile(content); err != nil {
			panic(err)
		}
	}
	if dirTest {
		fmt.Println("dir test")
		files, _ := WalkDir(content, "lua")

		for _, file := range files {
			if err := L.DoFile(file); err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "elatst",
		Usage:       "blockchain test.",
		Description: "With nodectl test, you could test blockchain.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "file, f",
				Usage: "test file",
			},
			cli.BoolFlag{
				Name:  "dir, d",
				Usage: "test dir",
			},
			cli.BoolFlag{
				Name:  "str, s",
				Usage: "test string",
			},

			cli.StringFlag{
				Name:  "content, c",
				Usage: "content",
			},
		},
		Action: elaTstAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "elatst")
			return cli.NewExitError("", 1)
		},
	}
}
