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

package predictor

import (
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/models"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	logger = logrus.WithField("module", "mpc.predictor")
)

// Model was trained out by a Learner,
// and participates in the multi-parts-calculation during prediction process
// If input different parts of a sample into Models on different mpc-nodes, you'll get final predicting result after some time of multi-parts-calculation
type Model interface {
	// Advance does calculation with local parts of samples and communicates with other nodes in cluster to predict outcomes
	// payload could be resolved by Model trained out by specific algorithm and samples
	// We'd better call the method asynchronously avoid blocking the main go-routine
	Advance(payload []byte) (*pb.PredictResponse, error)
}

// RpcHandler performs remote procedure calls to remote cluster nodes.
type RpcHandler interface {
	StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error)
}

// Callback contains some methods would be called when finish prediction
// such as to save the trained models and to stop a prediction task
// set into Predictor instance in initialization phase
type Callback interface {
	// SavePredictOut to persist prediction outcomes
	SavePredictOut(*pbCom.PredictTaskResult) error

	// StopTask to stop a prediction task
	// You'd better use it asynchronously to avoid deadlock
	StopTask(*pbCom.StopTaskRequest)
}

type PredictResponse struct {
	Resp *pb.PredictResponse
	Err  error
}

// Predictor manages Models, such as to create or to delete a model
// dispatches requests to different Models by taskId,
// keeps the number of Models in the proper range in order to avoid high memory usage
type Predictor struct {
	modelLimit int
	models     map[string]Model
	rpcHandler RpcHandler
	callback   Callback
	address    string
}

// NewModel creates a Model instance related to TaskId and stores it into Memory Storage
// keeps the number of Models in the proper range in order to avoid high memory usage
func (p *Predictor) NewModel(req *pbCom.StartTaskRequest) error {
	if p.modelLimit <= len(p.models) {
		err := errorx.New(errcodes.ErrCodeTooMuchTasks, "the number of tasks reached upper-limit %d", p.modelLimit)
		return err
	}

	taskId := req.TaskID
	if _, ok := p.modelExists(taskId); ok {
		err := errorx.New(errcodes.ErrCodeTaskExists, "task[%s] already exists ", taskId)
		return err
	}

	algo := req.GetParams().GetAlgo()
	params := req.GetParams().GetModelParams()
	file := req.GetFile()
	hosts := req.GetHosts()
	model, err := models.NewModel(taskId, p.address, algo, params, file, hosts, p.rpcHandler, p)
	if err != nil {
		return err
	}

	p.models[taskId] = model

	return nil
}

// DeleteModel deletes a task from Memory Storage
func (p *Predictor) DeleteModel(req *pbCom.StopTaskRequest) error {
	taskId := req.TaskID
	p.deleteModel(taskId)

	logger.WithField("taskId", taskId).Info("task deleted")
	return nil
}

// Predict dispatches requests to different Models by taskId
// resC returns the result, and couldn't be set with nil
func (p *Predictor) Predict(req *pb.PredictRequest, resC chan *PredictResponse) {
	setResult := func(resp *pb.PredictResponse, err error) {
		res := &PredictResponse{Resp: resp, Err: err}
		resC <- res
		close(resC)
	}

	taskId := req.TaskID
	mesg := req.GetPayload()
	if model, ok := p.modelExists(taskId); ok {
		go func() {
			resp, err := model.Advance(mesg)
			setResult(resp, err)
		}()
	} else {
		err := errorx.New(errcodes.ErrCodeParam, "task[%s] not exists ", taskId)
		setResult(nil, err)
	}
}

// SaveResult saves the prediction results (failed status or successful status) of samples for a Model
// and stops related task
func (p *Predictor) SaveResult(result *pbCom.PredictTaskResult) {
	err := p.callback.SavePredictOut(result)
	if err != nil {
		logger.WithField("taskId", result.TaskID).Errorf("failed to save outcomes[%v] and prediction result[%t], and error is[%s]",
			result.Outcomes, result.Success, err.Error())
	}

	logger.WithField("taskId", result.TaskID).Infof("Stop prediction task. And delete outcomes[%v] and prediction result[%t]",
		result.Outcomes, result.Success)

	req := &pbCom.StopTaskRequest{
		TaskID: result.TaskID,
		Params: &pbCom.TaskParams{
			TaskType: pbCom.TaskType_PREDICT,
		},
	}
	go p.callback.StopTask(req)
}

func (p *Predictor) modelExists(taskId string) (Model, bool) {
	if l, ok := p.models[taskId]; ok {
		return l, ok
	} else {
		return nil, false
	}
}

func (p *Predictor) deleteModel(taskId string) {
	delete(p.models, taskId)
}

// NewPredictor creates a Predictor instance,
// address indicates local mpc-node address
// modelLimit indicates the upper limit of the number of Models
// rh indicates the handler for rpc request sending
// cb indicates the callback methods called when finish prediction
func NewPredictor(address string, rh RpcHandler, cb Callback, modelLimit int) *Predictor {
	t := &Predictor{
		modelLimit: modelLimit,
		models:     make(map[string]Model, modelLimit),
		rpcHandler: rh,
		callback:   cb,
		address:    address,
	}

	return t
}
