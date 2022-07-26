# protoc-gen-auxo

**auxo** RPC support many codecs, if you use **Protocol Buffers** codec, you can generate contract codes from [Protocol Buffers](https://developers.google.com/protocol-buffers/docs/proto3) service definition files with **protoc-gen-auxo**.

## Install

Use `go get` to install the code generator:

```bash
go get github.com/cuigh/protoc-gen-auxo
```

You will also need:

* [protoc](https://github.com/google/protobuf), the protobuf compiler. You need version 3+.
* [protoc-gen-go](https://github.com/golang/protobuf/), the Go protobuf generator plugin. Get this with `go get github.com/golang/protobuf/protoc-gen-go`.

## Usage

Just like **grpc**:

```bash
protoc --go_out=. --auxo_out=. --go_opt=paths=source_relative --auxo_opt=paths=source_relative hello.proto
```

Service interfaces and client proxies were generated into a separate file `[name].auxo.go`:

```
hello.auxo.go
hello.pb.go
hello.proto
```