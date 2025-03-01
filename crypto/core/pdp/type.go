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

package pdp

import (
	"math/big"

	"github.com/cloudflare/bn256"
)

// PrivateKey PDP private key
type PrivateKey struct {
	X *big.Int
}

// PublicKey PDP public key
type PublicKey struct {
	P *bn256.G2
}

// CalculateSigmaIParams parameters required to calculate sigma_i for each segment
type CalculateSigmaIParams struct {
	Content []byte      // file content
	Index   *big.Int    // file segment index
	RandomV *big.Int    // a random V
	RandomU *big.Int    // a random U
	Privkey *PrivateKey // client private key
}

// ProofParams parameters required to generate proof
type ProofParams struct {
	Content  [][]byte    // file contents
	Indices  []*big.Int  // {i} index list
	RandomVs []*big.Int  // {v_i} random challenge number list
	Sigmas   []*bn256.G1 // {sigma_i} list in storage
}

// VerifyParams parameters required to verify a proof
type VerifyParams struct {
	Sigma    *bn256.G1  // sigma in proof
	Mu       *bn256.G1  // mu in proof
	RandomV  *big.Int   // a random V
	RandomU  *big.Int   // a random U
	Indices  []*big.Int // {i} index list
	RandomVs []*big.Int // {v_i} random challenge number list
	Pubkey   *PublicKey // client public key
}

// PrivateKeyToByte convert PDP private key to byes
func PrivateKeyToByte(privkey *PrivateKey) []byte {
	return privkey.X.Bytes()
}

// PrivateKeyFromByte retrieve PDP private key from byes
func PrivateKeyFromByte(privkey []byte) *PrivateKey {
	x := new(big.Int).SetBytes(privkey)
	return &PrivateKey{
		X: x,
	}
}

// PublicKeyToByte convert PDP public key to byes
func PublicKeyToByte(pubkey *PublicKey) []byte {
	return pubkey.P.Marshal()
}

// PublicKeyFromByte retrieve PDP public key from byes
func PublicKeyFromByte(pubkey []byte) (*PublicKey, error) {
	pub := new(bn256.G2)
	if _, err := pub.Unmarshal(pubkey); err != nil {
		return nil, err
	}
	return &PublicKey{
		P: pub,
	}, nil
}

// CalculateSigmaIParamsFromBytes retrieve CalculateSigmaIParams from bytes
func CalculateSigmaIParamsFromBytes(content, index, randomV, randomU, privkey []byte) CalculateSigmaIParams {
	return CalculateSigmaIParams{
		Content: content,
		Index:   new(big.Int).SetBytes(index),
		RandomV: new(big.Int).SetBytes(randomV),
		RandomU: new(big.Int).SetBytes(randomU),
		Privkey: PrivateKeyFromByte(privkey),
	}
}

// G1ToByte convert G1 point to byes
func G1ToByte(sigma *bn256.G1) []byte {
	return sigma.Marshal()
}

// G1FromByte retrieve G1 point from bytes
func G1FromByte(sigma []byte) (*bn256.G1, error) {
	s := new(bn256.G1)
	if _, err := s.Unmarshal(sigma); err != nil {
		return nil, err
	}
	return s, nil
}

// G1sFromBytes retrieve G1 point list from bytes
func G1sFromBytes(gs [][]byte) ([]*bn256.G1, error) {
	var ret []*bn256.G1
	for _, g := range gs {
		g1 := new(bn256.G1)
		if _, err := g1.Unmarshal(g); err != nil {
			return nil, err
		}
		ret = append(ret, g1)
	}
	return ret, nil
}

// IntListToBytes convert bit int list to bytes
func IntListToBytes(intList []*big.Int) [][]byte {
	var ret [][]byte
	for _, n := range intList {
		ret = append(ret, n.Bytes())
	}
	return ret
}

// IntListFromBytes retrieve bit int list from bytes
func IntListFromBytes(intList [][]byte) []*big.Int {
	var ret []*big.Int
	for _, n := range intList {
		ret = append(ret, new(big.Int).SetBytes(n))
	}
	return ret
}

// ProofParamsFromBytes retrieve ProofParams from bytes
func ProofParamsFromBytes(content, indices, randVs, sigmas [][]byte) (ProofParams, error) {
	s, err := G1sFromBytes(sigmas)
	if err != nil {
		return ProofParams{}, err
	}
	return ProofParams{
		Content:  content,
		Indices:  IntListFromBytes(indices),
		RandomVs: IntListFromBytes(randVs),
		Sigmas:   s,
	}, nil
}

// VerifyParamsFromBytes retrieve VerifyParams from bytes
func VerifyParamsFromBytes(sigma, mu, randV, randU, pubkey []byte, indices, randVs [][]byte) (VerifyParams, error) {
	s, err := G1FromByte(sigma)
	if err != nil {
		return VerifyParams{}, err
	}
	m, err := G1FromByte(mu)
	if err != nil {
		return VerifyParams{}, err
	}
	pub, err := PublicKeyFromByte(pubkey)
	if err != nil {
		return VerifyParams{}, err
	}

	return VerifyParams{
		Sigma:    s,
		Mu:       m,
		RandomV:  new(big.Int).SetBytes(randV),
		RandomU:  new(big.Int).SetBytes(randU),
		Indices:  IntListFromBytes(indices),
		RandomVs: IntListFromBytes(randVs),
		Pubkey:   pub,
	}, nil
}
