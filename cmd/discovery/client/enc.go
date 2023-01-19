package client

import (
	"encoding/json"
	"fmt"

	"github.com/kairos-io/kairos-challenger/pkg/constants"

	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kairos-io/tpm-helpers"
	"github.com/mudler/yip/pkg/utils"
	"github.com/pkg/errors"
)

func getPass(server string, partition *block.Partition) (string, bool, error) {
	msg, err := tpm.Get(server,
		tpm.WithAdditionalHeader("label", partition.Label),
		tpm.WithAdditionalHeader("name", partition.Name),
		tpm.WithAdditionalHeader("uuid", partition.UUID))
	if err != nil {
		return "", false, err
	}
	result := map[string]interface{}{}
	err = json.Unmarshal(msg, &result)
	if err != nil {
		return "", false, errors.Wrap(err, string(msg))
	}
	gen, generated := result["generated"]
	p, ok := result["passphrase"]
	if ok {
		return fmt.Sprint(p), generated && gen == constants.TPMSecret, nil
	}
	return "", false, errPartNotFound
}

func genAndStore(k Config) (string, error) {
	opts := []tpm.TPMOption{}
	if k.Kcrypt.Challenger.TPMDevice != "" {
		opts = append(opts, tpm.WithDevice(k.Kcrypt.Challenger.TPMDevice))
	}
	if k.Kcrypt.Challenger.CIndex != "" {
		opts = append(opts, tpm.WithIndex(k.Kcrypt.Challenger.CIndex))
	}

	// Generate a new one, and return it to luks
	rand := utils.RandomString(32)
	blob, err := tpm.EncodeBlob([]byte(rand))
	if err != nil {
		return "", err
	}
	nvindex := "0x1500000"
	if k.Kcrypt.Challenger.NVIndex != "" {
		nvindex = k.Kcrypt.Challenger.NVIndex
	}
	opts = append(opts, tpm.WithIndex(nvindex))
	return rand, tpm.StoreBlob(blob, opts...)
}

func localPass(k Config) (string, error) {
	index := "0x1500000"
	if k.Kcrypt.Challenger.NVIndex != "" {
		index = k.Kcrypt.Challenger.NVIndex
	}
	opts := []tpm.TPMOption{tpm.WithIndex(index)}
	if k.Kcrypt.Challenger.TPMDevice != "" {
		opts = append(opts, tpm.WithDevice(k.Kcrypt.Challenger.TPMDevice))
	}
	encodedPass, err := tpm.ReadBlob(opts...)
	if err != nil {
		// Generate if we fail to read from the assigned blob
		return genAndStore(k)
	}

	// Decode and give it back
	opts = []tpm.TPMOption{}
	if k.Kcrypt.Challenger.CIndex != "" {
		opts = append(opts, tpm.WithIndex(k.Kcrypt.Challenger.CIndex))
	}
	if k.Kcrypt.Challenger.TPMDevice != "" {
		opts = append(opts, tpm.WithDevice(k.Kcrypt.Challenger.TPMDevice))
	}
	pass, err := tpm.DecodeBlob(encodedPass, opts...)
	return string(pass), err
}
