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

// startTaskByIDCmd starts a confirmed task
var startTaskByIDCmd = &cobra.Command{
	Use:   "start",
	Short: "start the confirmed task",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}

		if err := client.StartTask(context.Background(), privateKey, id); err != nil {
			fmt.Printf("StartTask failed：%v\n", err)
			return
		}
		fmt.Println("ok")
	},
}

func init() {
	rootCmd.AddCommand(startTaskByIDCmd)

	startTaskByIDCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "requester private key hex string")
	startTaskByIDCmd.Flags().StringVarP(&id, "id", "i", "", "id of task to start, but only Ready and Failed tasks can be started")

	startTaskByIDCmd.MarkFlagRequired("privkey")
	startTaskByIDCmd.MarkFlagRequired("id")
}
