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

	"github.com/PaddlePaddle/PaddleDTX/dai/blockchain"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	requestClient "github.com/PaddlePaddle/PaddleDTX/dai/requester/client"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	files       string
	algorithm   string
	taskType    string
	taskName    string
	label       string
	labelName   string
	regMode     string
	regParam    float64
	alpha       float64
	amplitude   float64
	accuracy    uint64
	taskId      string
	description string
	psiLabel    string
	batchSize   uint64
)

// checkTaskPublishParams check mpc task parameters
// verify algorithm, taskType, regMode is legal
func checkTaskPublishParams() (pbCom.Algorithm, pbCom.TaskType, pbCom.RegMode, error) {
	var pAlgo pbCom.Algorithm
	var pType pbCom.TaskType
	var pRegMode pbCom.RegMode
	// task algorithm name check
	if algo, ok := blockchain.VlAlgorithmListName[algorithm]; ok {
		pAlgo = algo
	} else {
		return pAlgo, pType, pRegMode, errorx.New(errorx.ErrCodeParam, "algorithm only support linear-vl or logistic-vl")
	}
	// task type check
	if taskType, ok := blockchain.TaskTypeListName[taskType]; ok {
		pType = taskType
	} else {
		return pAlgo, pType, pRegMode, errorx.New(errorx.ErrCodeParam, "invalid task type: %s", taskType)
	}
	// task regMode check, no regularization if not set
	if mode, ok := blockchain.RegModeListName[regMode]; ok {
		pRegMode = mode
	} else {
		pRegMode = pbCom.RegMode_Reg_None
	}
	return pAlgo, pType, pRegMode, nil
}

// publishCmd publishes FL task
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "publish a task, can be a training task or a prediction task",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requestClient.GetRequestClient(configPath)
		if err != nil {
			fmt.Printf("GetRequestClient failed: %v\n", err)
			return
		}
		algo, taskType, regMode, err := checkTaskPublishParams()
		if err != nil {
			fmt.Printf("check task publish algoParam failed: %v\n", err)
			return
		}
		algorithmParams := pbCom.TaskParams{
			Algo:        algo,
			TaskType:    taskType,
			ModelTaskID: taskId,
			TrainParams: &pbCom.TrainParams{
				Label:     label,
				LabelName: labelName,
				RegMode:   regMode,
				RegParam:  regParam,
				Alpha:     alpha,
				Amplitude: amplitude,
				Accuracy:  int64(accuracy),
				BatchSize: int64(batchSize),
			},
		}

		taskID, err := client.Publish(context.Background(), requestClient.PublishOptions{
			PrivateKey:  privateKey,
			Files:       files,
			TaskName:    taskName,
			AlgoParam:   algorithmParams,
			Description: description,
			PSILabels:   psiLabel,
		})
		if err != nil {
			fmt.Printf("Publish task failed: %v\n", err)
			return
		}
		fmt.Println("TaskID:", taskID)
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVarP(&taskName, "name", "n", "", "task's name")
	publishCmd.Flags().StringVarP(&privateKey, "privkey", "k", "", "requester's private key hex string")
	publishCmd.Flags().StringVarP(&taskType, "type", "t", "", "task type, 'train' or 'predict'")
	publishCmd.Flags().StringVarP(&algorithm, "algorithm", "a", "", "algorithm assigned to task, 'linear-vl' and 'logistic-vl' are supported")
	publishCmd.Flags().StringVarP(&files, "files", "f", "", "sample files IDs with ',' as delimiter, like '123,456'")

	// optional params
	publishCmd.Flags().StringVarP(&label, "label", "l", "", "target feature for training task")
	publishCmd.Flags().StringVar(&labelName, "labelName", "", "target variable required in logistic-vl training")
	publishCmd.Flags().StringVarP(&psiLabel, "PSILabel", "p", "", "ID feature name list with ',' as delimiter, like 'id,id', required in vertical task")
	publishCmd.Flags().StringVarP(&taskId, "taskId", "i", "", "finished train task ID from which obtain the model, required for predict task")
	publishCmd.Flags().StringVar(&regMode, "regMode", "", "regularization mode required in train task, no regularization if not set, options are l1(L1-norm) and l2(L2-norm)")
	publishCmd.Flags().Float64Var(&regParam, "regParam", 0.1, "regularization parameter required in train task if set regMode")
	publishCmd.Flags().Float64Var(&alpha, "alpha", 0.1, "learning rate required in train task")
	publishCmd.Flags().Float64Var(&amplitude, "amplitude", 0.0001, "target difference of costs in two contiguous rounds that determines whether to stop training")
	publishCmd.Flags().Uint64Var(&accuracy, "accuracy", 10, "accuracy of homomorphic encryption")
	publishCmd.Flags().StringVarP(&description, "description", "d", "", "task description")
	publishCmd.Flags().Uint64VarP(&batchSize, "batchSize", "b", 4,
		"size of samples for one round of training loop, 0 for BGD(Batch Gradient Descent), non-zero for SGD(Stochastic Gradient Descent) or MBGD(Mini-Batch Gradient Descent)")

	publishCmd.MarkFlagRequired("name")
	publishCmd.MarkFlagRequired("privkey")
	publishCmd.MarkFlagRequired("type")
	publishCmd.MarkFlagRequired("algorithm")
	publishCmd.MarkFlagRequired("files")
}
