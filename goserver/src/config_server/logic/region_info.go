package logic

import (
	pb "common/proto/config"
	"config_server/logic/dao"
	"context"
)

func (s *Server) Info(ctx context.Context, req *pb.InfoReq) (*pb.InfoResp, error) {
	resp := &pb.InfoResp{}
	region := dao.ConfRegions{}
	//获取地址信息
	resp.Regions = region.GetRegion(ThisServer.GormDB, req.Id)
	return resp, nil
}
