package proto

//go:generate bash -c "mkdir -p ../build && rm ../build/* && protoc --go_out=../build --go_opt=paths=source_relative --go-grpc_out=../build --go-grpc_opt=paths=source_relative ./*.proto"
