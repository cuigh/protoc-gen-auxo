package main

import (
	"github.com/cuigh/auxo/app"
	"github.com/cuigh/auxo/app/flag"
	"github.com/cuigh/auxo/config"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	config.SetDefaultValue("banner", false)

	app.Name = "protoc-gen-auxo"
	app.Version = "0.21"
	app.Desc = `Code generator for auxo RPC

Usage: protoc --go_out=. --auxo_out=. --go_opt=paths=source_relative --auxo_opt=paths=source_relative hello.proto`
	app.Action = generate
	app.Flags.Register(flag.Help | flag.Version)
	app.Start()
}

func generate(_ *app.Context) error {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		g := NewGenerator(plugin)
		return g.Generate()
	})
	return nil
}
