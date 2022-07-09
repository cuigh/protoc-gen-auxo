package main

import (
	"io/ioutil"
	"os"

	"github.com/cuigh/auxo/app"
	"github.com/cuigh/auxo/app/flag"
	"github.com/cuigh/auxo/config"
	"github.com/cuigh/auxo/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	config.SetDefaultValue("banner", false)

	app.Name = "protoc-gen-auxo"
	app.Version = "0.2"
	app.Desc = `Code generator for auxo RPC

Usage: protoc --go_out=. --auxo_out=. hello.proto`
	app.Action = func(c *app.Context) error {
		return generate()
	}
	app.Flags.Register(flag.Help | flag.Version)
	app.Start()
}

func generate() error {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return errors.Wrap(err, "failed to read input")
	}

	req := &pluginpb.CodeGeneratorRequest{}
	err = proto.Unmarshal(data, req)
	if err != nil {
		return errors.Wrap(err, "failed to parse input proto")
	}

	g := NewGenerator(req)
	err = g.Generate()
	if err != nil {
		return errors.Wrap(err, "failed to generate output")
	}

	data, err = proto.Marshal(g.Response)
	if err != nil {
		return errors.Wrap(err, "failed to marshal output proto")
	}

	_, err = os.Stdout.Write(data)
	if err != nil {
		return errors.Wrap(err, "failed to write output")
	}
	return nil
}
