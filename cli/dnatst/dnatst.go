package dnatst

import (
	. "DNA_POW/cli/common"
	"DNA_POW/cli/dnatst/dnaapi"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	//. "DNA_POW/common"

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

func dnaTstAction(c *cli.Context) (err error) {
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
	L.PreloadModule("dnaapi", dnaapi.Loader)
	dnaapi.RegisterDataType(L)
	//dnaapi.RegisterAssetType(L)
	//dnaapi.RegisterBalanceTxInputType(L)
	//dnaapi.RegisterClientType(L)
	//dnaapi.RegisterFunctionCodeType(L)
	//dnaapi.RegisterTxAttributeType(L)
	//dnaapi.RegisterUTXOTxInputType(L)
	//dnaapi.RegisterTxOutputType(L)
	//dnaapi.RegisterBookKeeperType(L)
	//dnaapi.RegisterBookKeepingType(L)
	//dnaapi.RegisterCoinBaseType(L)
	//dnaapi.RegisterIssueAssetType(L)
	//dnaapi.RegisterTransferAssetType(L)
	//dnaapi.RegisterRegisterAssetType(L)
	//dnaapi.RegisterRecordType(L)
	//dnaapi.RegisterDataFileType(L)
	//dnaapi.RegisterPrivacyPayloadType(L)
	//dnaapi.RegisterDeployCodeType(L)
	//dnaapi.RegisterTransactionType(L)
	//dnaapi.RegisterBlockdataType(L)
	//dnaapi.RegisterBlockType(L)

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
		Name:        "dnatst",
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
		Action: dnaTstAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "dnatst")
			return cli.NewExitError("", 1)
		},
	}
}
