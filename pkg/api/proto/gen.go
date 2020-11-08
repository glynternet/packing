package proto

//go:generate bash -c "protoc --go_out=../build --go_opt=paths=source_relative --go-grpc_out=../build --go-grpc_opt=paths=source_relative ./*.proto"
