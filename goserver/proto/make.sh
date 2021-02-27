protoc -I./ --go_out=plugins=grpc:../src base.proto
protoc -I./ --go_out=plugins=grpc:../src  config.proto

#sed -i -e "s/,omitempty//g" ../src/common/proto/conf/conf.pb.go
#perl -i -pe "s/,omitempty//g" ../src/common/proto/conf/conf.pb.go

perl -i -pe "s/,omitempty//g" ../src/common/proto/config/config.pb.go