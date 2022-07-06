// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package utils

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/vm"
	"github.com/go-echarts/statsview"
	"github.com/go-echarts/statsview/viewer"

	"github.com/howeyc/gopass"
)

// GetPassword gets password from user input
func GetPassword() ([]byte, error) {
	fmt.Printf("Password:")
	return gopass.GetPasswd()
}

// GetConfirmedPassword gets double confirmed password from user input
func GetConfirmedPassword() ([]byte, error) {
	fmt.Printf("Password:")
	first, err := gopass.GetPasswd()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Re-enter Password:")
	second, err := gopass.GetPasswd()
	if err != nil {
		return nil, err
	}
	if len(first) != len(second) {
		fmt.Println("Unmatched Password")
		os.Exit(1)
	}
	for i, v := range first {
		if v != second[i] {
			fmt.Println("Unmatched Password")
			os.Exit(1)
		}
	}
	return first, nil
}

func StartPProf(port uint32, host string) {
	listenAddr := net.JoinHostPort("", strconv.FormatUint(
		uint64(port), 10))
	viewer.SetConfiguration(viewer.WithMaxPoints(100),
		viewer.WithInterval(300000),
		viewer.WithAddr(listenAddr),
		viewer.WithLinkAddr(host))
	mgr := statsview.New()
	mgr.Start()
}

func FileExisted(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func StringExisted(src []string, check string) bool {
	for _, ar := range src {
		if ar == check {
			return true
		}
	}
	return false
}

// CopyStringSet copy the src map's key, and return the dst map.
func CopyStringSet(src map[string]struct{}) (dst map[string]struct{}) {
	dst = map[string]struct{}{}
	for k := range src {
		dst[k] = struct{}{}
	}
	return
}

// CopyStringMap copy the src map's key and value, and return the dst map.
func CopyStringMap(src map[string]string) (dst map[string]string) {
	dst = map[string]string{}
	for k, v := range src {
		p := v
		dst[k] = p
	}
	return
}

func SerializeStringMap(w io.Writer, smap map[string]string) (err error) {
	if err = common.WriteVarUint(w, uint64(len(smap))); err != nil {
		return
	}
	for k, v := range smap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}

		if err = common.WriteVarString(w, v); err != nil {
			return
		}
	}
	return
}

func DeserializeStringMap(r io.Reader) (smap map[string]string, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	smap = make(map[string]string)
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		var v string
		if v, err = common.ReadVarString(r); err != nil {
			return
		}
		smap[k] = v
	}
	return
}

func SerializeStringSet(w io.Writer, vmap map[string]struct{}) (err error) {
	if err = common.WriteVarUint(w, uint64(len(vmap))); err != nil {
		return
	}
	for k := range vmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}
	}
	return
}

func DeserializeStringSet(r io.Reader) (vmap map[string]struct{}, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	vmap = make(map[string]struct{})
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}
		vmap[k] = struct{}{}
	}
	return
}

func GetAddressByCode(code []byte) (string, error) {
	programHash, err := GetProgramHashByCode(code)
	if err != nil {
		return "", err
	}
	address, err := programHash.ToAddress()
	if err != nil {
		return "", err
	}
	return address, nil
}

func GetProgramHashByCode(code []byte) (*common.Uint168, error) {
	signType, err := crypto.GetScriptType(code)
	if err != nil {
		return nil, err
	}
	if signType == vm.CHECKSIG {
		ct, err := contract.CreateStandardContractByCode(code)
		if err != nil {
			return nil, err
		}
		return ct.ToProgramHash(), nil

	} else if signType == vm.CHECKMULTISIG {
		ct, err := contract.CreateMultiSigContractByCode(code)
		if err != nil {
			return nil, err
		}
		return ct.ToProgramHash(), nil
	} else {
		return nil, errors.New("invalid code type")
	}
}
