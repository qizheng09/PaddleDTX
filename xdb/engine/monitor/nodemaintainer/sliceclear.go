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

package nodemaintainer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// sliceclear cleans expired encrypted slices
func (m *NodeMaintainer) sliceclear(ctx context.Context) {
	pubkey := ecdsa.PublicKeyFromPrivateKey(m.localNode.PrivateKey)
	clearKey := m.getClearKey(pubkey)

	l := logger.WithField("runner", "slice clear loop")
	node, err := m.blockchain.GetNode(ctx, []byte(pubkey.String()))
	if err != nil {
		l.WithError(err).Warn("failed to get node info")
		return
	}
	defer l.Info("slice clear stopped")

	ticker := time.NewTicker(m.fileClearInterval)
	defer ticker.Stop()

	m.doneSliceClearC = make(chan struct{})
	defer close(m.doneSliceClearC)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		latestTime := time.Now().UnixNano()
		if node.RegTime > (latestTime - m.fileRetainInterval.Nanoseconds()) {
			l.Info("node register time is later than slice retain time, no slice to clear")
			continue
		}

		startTime, endTime, err := m.getExpireRangeTime(ctx, pubkey.String(), clearKey, latestTime, node.RegTime)
		if err != nil {
			l.WithError(err).Warn("failed to get expire range time")
			continue
		}
		if startTime == endTime {
			l.Info("startTime and endTime too close")
			continue
		}

		opt := &blockchain.ListNodeSliceOptions{
			Target:    []byte(pubkey.String()),
			StartTime: startTime,
			EndTime:   endTime,
		}

		sliceList, err := m.blockchain.ListNodesExpireSlice(ctx, opt)
		if err != nil {
			l.WithError(err).Warn("failed to get expire slice")
			continue
		}
		var deleteSlice []string
		var delete bool
		for _, slice := range sliceList {
			exist, _ := m.sliceStorage.Exist(slice)
			// 如果存在，删除切片
			if exist {
				deleteSlice = append(deleteSlice, slice)
				delete, err = m.sliceStorage.Delete(slice)
				if !delete {
					break
				}
			}
		}
		if err != nil {
			l.WithError(err).Warn("failed to delete node slice")
			continue
		}
		err = m.sliceStorage.SaveAndUpdate(clearKey, strconv.FormatInt(endTime, 10))
		if err != nil {
			l.WithError(err).Warn("failed to update clear slice time ")
		}

		l.WithFields(logrus.Fields{
			"start_time":     time.Unix(0, startTime).Format("2006-01-02 15:04:05"),
			"end_time":       time.Unix(0, endTime).Format("2006-01-02 15:04:05"),
			"update_at":      time.Now().Format("2006-01-02 15:04:05"),
			"dslice_id_list": strings.Join(deleteSlice, ","),
		}).Info("success to clear slice of node")
	}
}

func (m *NodeMaintainer) getExpireRangeTime(ctx context.Context, pubkey, clearKey string, latestTime,
	regTime int64) (int64, int64, error) {
	var startTime, endTime int64
	exist, _ := m.sliceStorage.Exist(clearKey)
	if !exist {
		if err := m.sliceStorage.SaveAndUpdate(clearKey, strconv.FormatInt(regTime, 10)); err != nil {
			return 0, 0, errorx.Wrap(err, "failed to save and update slice")
		}
		endTime = m.getEndExpireTime(regTime, latestTime)
		return regTime, endTime, nil
	}
	ftime, err := m.sliceStorage.LoadStr(clearKey)
	if err != nil {
		return 0, 0, errorx.Wrap(err, "failed to load expire time")
	}
	if startTime, err = strconv.ParseInt(ftime, 10, 64); err != nil {
		return 0, 0, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parse expire time")
	}
	endTime = m.getEndExpireTime(startTime, latestTime)
	return startTime, endTime, nil
}

func (m *NodeMaintainer) getClearKey(pubkey ecdsa.PublicKey) string {
	suid := fmt.Sprintf("%x-%x-%x-%x-%x", pubkey[0:4], pubkey[4:6], pubkey[6:8], pubkey[8:10], pubkey[10:16])
	return suid
}

func (m *NodeMaintainer) getEndExpireTime(startTime, latestTime int64) (endTime int64) {
	interH := m.fileRetainInterval.Nanoseconds()
	if latestTime-interH < startTime {
		endTime = startTime
	} else {
		endTime = latestTime - interH
	}
	return endTime
}
