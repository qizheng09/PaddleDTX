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

package task

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
)

var (
	output string
)

// getPredictResCmd gets predict task result from Executor
var getPredictResCmd = &cobra.Command{
	Use:   "result",
	Short: "get predict task result from executor node",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}

		if err := client.GetPredictResult(context.Background(), privateKey, id, output); err != nil {
			fmt.Printf("GetPredictResult failed：%v\n", err)
			return
		}

		fmt.Println("ok")
	},
}

func init() {
	rootCmd.AddCommand(getPredictResCmd)

	getPredictResCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "requester private key hex string")
	getPredictResCmd.Flags().StringVarP(&id, "id", "i", "", "prediction task id")
	getPredictResCmd.Flags().StringVarP(&output, "output", "o", "", "file to store prediction outcomes")

	getPredictResCmd.MarkFlagRequired("privkey")
	getPredictResCmd.MarkFlagRequired("id")
	getPredictResCmd.MarkFlagRequired("output")
}
