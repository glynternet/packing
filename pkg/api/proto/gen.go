package proto

// apt install protobuf-compiler protoc-gen-go
// go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
//go:generate bash -c "mkdir -p ../build && protoc --go_out=../build --go_opt=paths=source_relative --go-grpc_out=../build --go-grpc_opt=paths=source_relative ./*.proto"
