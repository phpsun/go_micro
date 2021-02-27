package util

import (
	"google.golang.org/grpc/codes"
	grpcerr "google.golang.org/grpc/status"
)

func MakeGrpcError(code int32, msg string) error {
	return grpcerr.Error(codes.Code(code), msg)
}

func FromGrpcError(err error) (int32, string) {
	s, _ := grpcerr.FromError(err)
	return int32(s.Code()), s.Message()
}
