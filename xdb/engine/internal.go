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

package engine

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	ctype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/challenger/merkle/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/encryptor"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/merkle"
)

// packChainFile rearranges the encrypted slices and calculates the digest that will be sent onto blockchain later
func (e *Engine) packChainFile(fileID, challengAlgorithm string, opt types.WriteOptions, originalSlices slicer.SliceMetas,
	originalLen int, encryptedSlices []encryptor.EncryptedSlice, pdp types.PDP) (blockchain.File, error) {

	sliceIdxMap := make(map[string]int)
	chainSlices := make([]blockchain.PublicSliceMeta, 0, len(originalSlices))

	// encryptedSlices in random order
	randEncSlices := rearrangeEncSlices(encryptedSlices)
	for _, s := range randEncSlices { // dis-ordered
		sps := blockchain.PublicSliceMeta{
			ID:         s.SliceID,
			Length:     s.Length,
			NodeID:     s.NodeID,
			CipherHash: s.CipherHash,
		}
		if challengAlgorithm == types.PDPChallengAlgorithm {
			// denote slice index for each node (for pdp challenge)
			nodeStr := base64.StdEncoding.EncodeToString(s.NodeID)
			if _, ok := sliceIdxMap[nodeStr]; !ok {
				sliceIdxMap[nodeStr] = 1
			} else {
				sliceIdxMap[nodeStr] += 1
			}
			idx := big.NewInt(int64(sliceIdxMap[nodeStr]))
			sigmaI, err := xchainClient.CalculatePDPSigmaI(s.CipherText, idx.Bytes(), pdp.RandV, pdp.RandU, pdp.PdpPrivkey)
			if err != nil {
				return blockchain.File{}, errorx.Wrap(err, "CalculatePDPSigmaI failed")
			}
			sps.SliceIdx = sliceIdxMap[nodeStr]
			sps.SigmaI = sigmaI
		}
		chainSlices = append(chainSlices, sps)
	}
	structure, err := e.packChainFileStructure(originalSlices)
	if err != nil {
		return blockchain.File{}, errorx.Wrap(err, "failed to pack chain file structure")
	}

	merkleRoot := calculateMerkleRoot(originalSlices)
	owner, _ := hex.DecodeString(opt.User)

	chainFile := blockchain.File{
		ID:          fileID,
		Name:        opt.FileName,
		Description: opt.Description,
		Namespace:   opt.Namespace,
		Owner:       owner,
		Length:      uint64(originalLen),
		MerkleRoot:  merkleRoot,
		Slices:      chainSlices,
		Structure:   structure,
		PublishTime: time.Now().UnixNano(),
		ExpireTime:  opt.ExpireTime,
		Ext:         []byte(opt.Extra),
	}
	if challengAlgorithm == types.PDPChallengAlgorithm {
		chainFile.PdpPubkey = pdp.PdpPubkey
		chainFile.RandU = pdp.RandU
		chainFile.RandV = pdp.RandV
	}

	return chainFile, nil
}

// packChainFileStructure pack file private structure and encrypt it
func (e *Engine) packChainFileStructure(originalSlices slicer.SliceMetas) ([]byte, error) {
	structure := make(blockchain.FileStructure, 0, len(originalSlices))
	for _, s := range originalSlices {
		structure = append(structure, blockchain.PrivateSliceMeta{
			SliceID:   s.ID,
			PlainHash: s.Hash,
		})
	}
	raw, err := structure.Marshal()
	if err != nil {
		return nil, err
	}
	// encrypt structure
	encStruct, err := e.encryptor.Encrypt(context.TODO(), bytes.NewReader(raw), &encryptor.EncryptOptions{})
	if err != nil {
		return nil, err
	}
	return encStruct.CipherText, err
}

// recoverChainFileStructure get file structure from blockchain and decrypt it
func (e *Engine) recoverChainFileStructure(bs []byte) (blockchain.FileStructure, error) {
	// decrypt structure
	decStruct, err := e.encryptor.Recover(context.TODO(), bytes.NewReader(bs), &encryptor.RecoverOptions{})
	if err != nil {
		return nil, err
	}

	var fs blockchain.FileStructure
	if err := fs.Parse(decStruct); err != nil {
		return fs, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parse file structure")
	}

	return fs, nil
}

// getMerkleChallengerRange get merkle challenge material for challenge request later
func (e *Engine) getMerkleChallengerRange(fileID string, es encryptor.EncryptedSlice,
	expireTime int64, merkleMaterialQueue chan<- ctype.Material) (err error) {
	// get challenging materials
	ti := e.monitor.challengingMonitor.RequestInterval.Nanoseconds()
	return common.GetAddFileMCRange(e.challenger, fileID, es, expireTime, time.Now().UnixNano(), ti, merkleMaterialQueue)
}

func calculateMerkleRoot(slices slicer.SliceMetas) []byte {
	hashes := make([][]byte, 0, len(slices))
	for _, s := range slices {
		hashes = append(hashes, s.Hash)
	}

	return merkle.GetMerkleRoot(hashes)
}

// calculateFileMaxStructSize calculate a file's max structure size that will be saved on blockchain
func calculateFileMaxStructSize(slicesNum, replica int) int {
	// every slice size < 400
	slicesSize := slicesNum * replica * 400
	// every file struct size < 200
	structSize := slicesNum * 200
	// base file size, not incule silces and structure < 1000
	fileBaseSize := 1000
	// file json marshal
	fileConvertSize := 1000
	return fileBaseSize + structSize + slicesSize + fileConvertSize
}

// rearrangeEncSlices arrange encrypted slices in random order
func rearrangeEncSlices(encryptedSlices []encryptor.EncryptedSlice) []encryptor.EncryptedSlice {
	num := len(encryptedSlices)
	for i := 0; i < num; i++ {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(num)))
		k, _ := rand.Int(rand.Reader, big.NewInt(int64(num)))
		encryptedSlices[j.Int64()], encryptedSlices[k.Int64()] = encryptedSlices[k.Int64()], encryptedSlices[j.Int64()]
	}
	return encryptedSlices
}
