// Copyright (c) 2021 PaddlePaddle Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package soft

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/aes"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// SoftEncryptor encrypts data or decrypts encoded data
type SoftEncryptor struct {
	password string
}

// New creat SoftEncryptor by "password" configuration
func New(conf *config.SoftEncryptorConf) (*SoftEncryptor, error) {
	if len(conf.Password) == 0 {
		return nil, errorx.New(errorx.ErrCodeConfig, "missing password")
	}

	se := &SoftEncryptor{
		password: conf.Password,
	}

	return se, nil
}

// Encrypt derive key using nodeID and slice ID, then encrypt content using AES-GCM
func (se *SoftEncryptor) Encrypt(ctx context.Context, r io.Reader, opt *encryptor.EncryptOptions) (
	encryptor.EncryptedSlice, error) {

	key := se.getKey(opt.SliceID, opt.NodeID)

	salt := append(append([]byte{}, []byte(opt.SliceID)...), opt.NodeID...)
	nonce := hash.HashUsingSha256(salt)[:12]

	aesKey := aes.AESKey{
		Key:   key,
		Nonce: nonce,
	}

	plaintext, err := ioutil.ReadAll(r)
	if err != nil {
		return encryptor.EncryptedSlice{},
			errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read")
	}

	ciphertext, err := aes.EncryptUsingAESGCM(aesKey, plaintext, nil)
	if err != nil {
		return encryptor.EncryptedSlice{}, errorx.Wrap(err, "failed to encrypt")
	}
	h := hash.HashUsingSha256(ciphertext)

	es := encryptor.EncryptedSlice{
		EncryptedSliceMeta: encryptor.EncryptedSliceMeta{
			SliceID:    opt.SliceID,
			NodeID:     opt.NodeID,
			CipherHash: h,
			Length:     uint64(len(ciphertext)),
		},
		CipherText: ciphertext,
	}

	return es, nil
}

// Encrypt derive key using nodeID and slice ID, then decrypt content using AES-GCM
func (se *SoftEncryptor) Recover(ctx context.Context, r io.Reader, opt *encryptor.RecoverOptions) (
	[]byte, error) {

	key := se.getKey(opt.SliceID, opt.NodeID)

	salt := append(append([]byte{}, []byte(opt.SliceID)...), opt.NodeID...)
	nonce := hash.HashUsingSha256(salt)[:12]

	aesKey := aes.AESKey{
		Key:   key,
		Nonce: nonce,
	}

	ciphertext, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read")
	}

	plaintext, err := aes.DecryptUsingAESGCM(aesKey, ciphertext, nil)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to decrypt")
	}

	return plaintext, nil
}
