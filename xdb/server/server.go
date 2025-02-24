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

package server

import (
	"context"
	"io"

	"github.com/kataras/iris/v12"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	etype "github.com/PaddlePaddle/PaddleDTX/xdb/engine/types"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

// Handler defines all apis exposed
type Handler interface {
	Write(context.Context, etype.WriteOptions, io.Reader) (etype.WriteResponse, error)
	Read(context.Context, etype.ReadOptions) (io.ReadCloser, error)

	ListFiles(context.Context, etype.ListFileOptions) ([]blockchain.File, error)
	ListExpiredFiles(context.Context, etype.ListFileOptions) ([]blockchain.File, error)
	GetFileByID(ctx context.Context, id string) (file blockchain.FileH, err error)
	GetFileByName(ctx context.Context, owner []byte, ns, name string) (file blockchain.FileH, err error)
	UpdateFileExpireTime(ctx context.Context, opt etype.UpdateFileEtimeOptions) error
	AddFileNs(ctx context.Context, opt etype.AddNsOptions) error
	UpdateNsReplica(ctx context.Context, opt etype.UpdateNsOptions) error
	ListFileNs(ctx context.Context, opt etype.ListNsOptions) ([]blockchain.Namespace, error)
	GetNsByName(ctx context.Context, owner []byte, name string) (blockchain.NamespaceH, error)
	GetFileSysHealth(ctx context.Context, owner []byte) (blockchain.FileSysHealth, error)
	GetChallengeById(ctx context.Context, id string) (blockchain.Challenge, error)
	GetChallenges(ctx context.Context, opt blockchain.ListChallengeOptions) ([]blockchain.Challenge, error)

	Push(context.Context, etype.PushOptions, io.Reader) (etype.PushResponse, error)
	Pull(context.Context, etype.PullOptions) (io.ReadCloser, error)

	AddNode(context.Context, etype.AddNodeOptions) error
	ListNodes(context.Context) (blockchain.Nodes, error)
	GetNode(context.Context, []byte) (blockchain.Node, error)
	GetHeartbeatNum(context.Context, []byte, int64) (int, int, error)
	GetNodeHealth(context.Context, []byte) (string, error)
	NodeOffline(context.Context, etype.NodeOfflineOptions) error
	NodeOnline(context.Context, etype.NodeOnlineOptions) error
	GetSliceMigrateRecords(ctx context.Context, opt *blockchain.NodeSliceMigrateOptions) (string, error)
}

// Server http server
type Server struct {
	app *iris.Application

	listenAddr string
	handler    Handler
}

// New initiate Server
func New(listenAddress string, h Handler) (*Server, error) {
	app := iris.New()
	if listenAddress == "" {
		return nil, errorx.New(errorx.ErrCodeConfig, "misssing config: listenAddress")
	}

	server := &Server{
		app:        app,
		listenAddr: listenAddress,
		handler:    h,
	}
	return server, nil
}

func (s *Server) setRoute(serverType string) (err error) {
	v1 := s.app.Party("/v1")
	nodeParty := v1.Party("/node")
	switch serverType {
	// storage
	case config.NodeTypeStorage:
		sliceParty := v1.Party("/slice")
		sliceParty.Post("/push", s.push)
		sliceParty.Get("/pull", s.pull)

		nodeParty.Post("/add", s.addNode)
		nodeParty.Get("/list", s.listNodes)
		nodeParty.Get("/get", s.getNode)
		nodeParty.Get("/health", s.getNodeHealth)
		nodeParty.Post("/offline", s.nodeOffline)
		nodeParty.Post("/online", s.nodeOnline)
		nodeParty.Get("/getmrecord", s.getMRecord)
		nodeParty.Get("/gethbnum", s.getHeartbeatNum)
	// dataOwner
	case config.NodeTypeDataOwner:
		fileParty := v1.Party("/file")
		fileParty.Post("/write", s.write)
		fileParty.Get("/read", s.read)
		fileParty.Get("/list", s.listFiles)
		fileParty.Get("/listexp", s.listExpiredFiles)
		fileParty.Get("/getbyid", s.getFileByID)
		fileParty.Get("/getbyname", s.getFileByName)
		fileParty.Post("/updatexptime", s.updateFileExpireTime)
		fileParty.Post("/addns", s.addFileNs)
		fileParty.Post("/ureplica", s.updateNsReplica)
		fileParty.Get("/listns", s.listFileNs)
		fileParty.Get("/getns", s.getNsByName)
		fileParty.Get("/getsyshealth", s.getSysHealth)

		nodeParty.Get("/list", s.listNodes)
		nodeParty.Get("/get", s.getNode)
		nodeParty.Get("/health", s.getNodeHealth)
		nodeParty.Get("/getmrecord", s.getMRecord)
		nodeParty.Get("/gethbnum", s.getHeartbeatNum)

		challParty := v1.Party("/challenge")
		challParty.Get("/getbyid", s.getChallengeById)
		challParty.Get("/toprove", s.getToProveChallenges)
		challParty.Get("/proved", s.getProvedChallenges)
		challParty.Get("/failed", s.getFailedChallenges)
	default:
		err = errorx.New(errorx.ErrCodeConfig, "wrong config: server.server-type")
	}
	s.app.OnAnyErrorCode(func(ictx iris.Context) {
		responseError(ictx, errorx.New(errorx.ErrCodeNotFound, "request url not found"))
	})
	return err
}

// Serve runs and blocks current routine
func (s *Server) Serve(ctx context.Context) error {
	if err := s.setRoute(ctx.Value("server-type").(string)); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		logrus.Info("server stops ...")
		s.app.Shutdown(context.TODO())
	}()

	logrus.Infof("server starts, and listens port %s", s.listenAddr)
	if err := s.app.Listen(s.listenAddr); err != nil {
		//error occurs when start server
		return err
	}

	return ctx.Err()
}
