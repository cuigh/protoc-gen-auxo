package main

import (
	"bytes"
	"fmt"
	"github.com/cuigh/auxo/app"
	"github.com/cuigh/auxo/data"
	"github.com/cuigh/auxo/ext/texts"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	// tag numbers in FileDescriptorProto
	packagePath = 2 // package
	messagePath = 4 // message_type
	enumPath    = 5 // enum_type
	servicePath = 6 // service
	// tag numbers in DescriptorProto
	messageFieldPath   = 2 // field
	messageMessagePath = 3 // nested_type
	messageEnumPath    = 4 // enum_type
	messageOneofPath   = 8 // oneof_decl
	// tag numbers in ServiceDescriptorProto
	serviceNamePath    = 1 // name
	serviceMethodPath  = 2 // method
	serviceOptionsPath = 3 // options
	// tag numbers in MethodDescriptorProto
	methodNamePath   = 1 // name
	methodInputPath  = 2 // input_type
	methodOutputPath = 3 // output_type
)

type Generator struct {
	plugin *protogen.Plugin
	types  map[string]*TypeInfo
	b      *Builder
}

func NewGenerator(plugin *protogen.Plugin) *Generator {
	g := &Generator{
		plugin: plugin,
		types:  make(map[string]*TypeInfo),
		b:      &Builder{},
	}
	g.initTypes()
	return g
}

func (g *Generator) initTypes() {
	for _, file := range g.plugin.Request.ProtoFile {
		for _, t := range file.GetMessageType() {
			g.initMessageTypes(file, t)
		}
		for _, t := range file.GetEnumType() {
			ti := newTypeInfo(file.GetPackage(), file.GetOptions().GetGoPackage(), t.GetName())
			g.types[ti.ProtoName()] = ti
		}
	}
}

func (g *Generator) initMessageTypes(file *descriptorpb.FileDescriptorProto, t *descriptorpb.DescriptorProto) {
	ti := newTypeInfo(file.GetPackage(), file.GetOptions().GetGoPackage(), t.GetName())
	g.types[ti.ProtoName()] = ti

	for _, nt := range t.NestedType {
		g.initMessageTypes(file, nt)
	}
	for _, et := range t.EnumType {
		ti := newTypeInfo(file.GetPackage(), file.GetOptions().GetGoPackage(), et.GetName())
		g.types[ti.ProtoName()] = ti
	}
}

func (g *Generator) Generate() error {
	for _, file := range g.plugin.Files {
		if len(file.Services) == 0 {
			continue
		}

		g.b.Reset()
		g.generateHeader(file.Proto)
		g.generateImports(file)
		g.generateVariables(file.Proto)
		for i, service := range file.Proto.GetService() {
			g.b.Line()
			g.generateService(file.Proto, service, i)
		}

		filename := file.GeneratedFilenamePrefix + ".auxo.go"
		f := g.plugin.NewGeneratedFile(filename, file.GoImportPath)
		_, _ = f.Write(g.b.buf.Bytes())
	}
	return nil
}

func (g *Generator) getProtoFiles() []*descriptorpb.FileDescriptorProto {
	files := make([]*descriptorpb.FileDescriptorProto, 0)
	for _, name := range g.plugin.Request.GetFileToGenerate() {
		for _, file := range g.plugin.Request.GetProtoFile() {
			if file.GetName() == name {
				files = append(files, file)
			}
		}
	}
	return files
}

func (g *Generator) findLocation(file *descriptorpb.FileDescriptorProto, path []int32) *descriptorpb.SourceCodeInfo_Location {
	if file.SourceCodeInfo != nil {
		for _, loc := range file.SourceCodeInfo.Location {
			if g.pathEqual(path, loc.Path) {
				return loc
			}
		}
	}
	return nil
}

func (g *Generator) pathEqual(path1, path2 []int32) bool {
	if len(path1) != len(path2) {
		return false
	}
	for i, v := range path1 {
		if path2[i] != v {
			return false
		}
	}
	return true
}

// todo: support generate models
//func (g *Generator) generateModel(file *descriptorpb.FileDescriptorProto) *plugin.CodeGeneratorResponse_File {
//}

func (g *Generator) generateHeader(file *descriptorpb.FileDescriptorProto) {
	pkg := filepath.Base(file.GetOptions().GetGoPackage())
	if pkg == "" {
		pkg = file.GetPackage()
	}
	g.b.Line("// Code generated by protoc-gen-auxo ", app.Version, ", DO NOT EDIT.")
	g.b.Line("// source: ", file.GetName())
	g.b.Line()
	g.b.Line(`package `, pkg)
	g.b.Line()
}

func (g *Generator) generateImports(file *protogen.File) {
	g.b.Line(`import (
	"context"

	"github.com/cuigh/auxo/net/rpc"
)`)
	g.b.Line()
}

func (g *Generator) generateVariables(file *descriptorpb.FileDescriptorProto) {
	server := file.GetPackage()
	if i := strings.LastIndex(server, "."); i > 0 {
		server = server[:i]
	}
	server = strings.ReplaceAll(server, "_", "-")

	max := 0
	vars := data.Options{}
	for _, service := range file.GetService() {
		name := texts.Rename(service.GetName(), texts.Camel)
		if l := len(name); l > max {
			max = l
		}
		vars = append(vars, data.Option{Name: name, Value: fmt.Sprintf(`&%sClient{rpc.LazyClient{Name: "%s"}}`, name, server)})
	}

	g.b.Line("var (")
	g.b.Align(vars, "\t", max)
	g.b.Line(")")
}

func (g *Generator) generateService(file *descriptorpb.FileDescriptorProto, service *descriptorpb.ServiceDescriptorProto, index int) {
	path := []int32{servicePath, int32(index)}
	loc := g.findLocation(file, path)
	g.generateComments(loc, "")

	// interface
	g.b.Write("type ", service.GetName(), " interface {")
	if c := loc.GetTrailingComments(); c != "" {
		g.b.Comment(c, " ")
	} else {
		g.b.Line()
	}
	for i, method := range service.GetMethod() {
		path = []int32{servicePath, int32(index), serviceMethodPath, int32(i)}
		loc := g.findLocation(file, path)
		g.generateComments(loc, "\t")
		g.b.Format("\t%s(context.Context, *%s) (*%s, error)",
			method.GetName(),
			g.goTypeName(file.GetPackage(), method.GetInputType()),
			g.goTypeName(file.GetPackage(), method.GetOutputType()))
		if c := loc.GetTrailingComments(); c != "" {
			g.b.Comment(c, " ")
		} else {
			g.b.Line()
		}
	}
	g.b.Line("}")
	g.b.Line()

	// getter
	name := texts.Rename(service.GetName(), texts.Camel)
	g.b.Format(`func Get%s() %s {
	return %s
}`, service.GetName(), service.GetName(), name)
	g.b.Line()
	g.b.Line()

	// implementation
	g.b.Line("type ", name, "Client struct {")
	g.b.Line("\trpc.LazyClient")
	g.b.Line("}")
	for _, method := range service.Method {
		g.b.Line()
		g.generateMethod(file, service, method)
	}
}

func (g *Generator) generateComments(loc *descriptorpb.SourceCodeInfo_Location, prefix string) {
	if loc != nil {
		if comments := loc.GetLeadingDetachedComments(); len(comments) > 0 {
			for _, c := range comments {
				g.b.Comment(c, prefix)
			}
			g.b.Line()
		}
		g.b.Comment(loc.GetLeadingComments(), prefix)
	}
}

func (g *Generator) generateMethod(file *descriptorpb.FileDescriptorProto, service *descriptorpb.ServiceDescriptorProto,
	method *descriptorpb.MethodDescriptorProto) {
	tpl := `func (s *${Type}) ${Method}(ctx context.Context, req *${Request}) (*${Response}, error) {
	c, err := s.Try()
	if err != nil {
		return nil, err
	}

	resp := new(${Response})
	err = c.Call(ctx, "${Service}", "${Method}", []interface{}{req}, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}`
	args := map[string]string{
		"Service":  service.GetName(),
		"Type":     texts.Rename(service.GetName(), texts.Camel) + "Client",
		"Method":   method.GetName(),
		"Request":  g.goTypeName(file.GetPackage(), method.GetInputType()),
		"Response": g.goTypeName(file.GetPackage(), method.GetOutputType()),
	}
	g.b.Expand(tpl, args)
}

// goTypeName returns the Go type name, including it's package prefix.
func (g *Generator) goTypeName(protoPkg, protoName string) string {
	if ti, ok := g.types[protoName]; ok {
		return ti.RelativeName(protoPkg)
	}
	panic("unknown type: " + protoName)
}

type TypeInfo struct {
	ProtoPackage string
	Name         string
	FullName     string
}

func newTypeInfo(protoPkg, goPkg, protoName string) *TypeInfo {
	i := strings.LastIndex(protoName, ".")
	if goPkg == "" {
		goPkg = strings.Replace(protoPkg, ".", "_", -1)
	}
	ti := &TypeInfo{
		ProtoPackage: protoPkg,
		Name:         protoName[i+1:],
	}
	ti.FullName = goPkg + "." + ti.Name
	return ti
}

func (t *TypeInfo) ProtoName() string {
	return "." + t.ProtoPackage + "." + t.Name
}

func (t *TypeInfo) RelativeName(protoPkg string) string {
	if t.ProtoPackage == protoPkg {
		return t.Name
	}
	return t.FullName
}

type Builder struct {
	buf bytes.Buffer
}

func (b *Builder) Reset() {
	b.buf.Reset()
}

func (b *Builder) String() string {
	return b.buf.String()
}

func (b *Builder) Write(args ...string) {
	for _, v := range args {
		b.buf.WriteString(v)
	}
}

func (b *Builder) Line(args ...string) {
	for _, v := range args {
		b.buf.WriteString(v)
	}
	b.buf.WriteByte('\n')
}

func (b *Builder) Format(format string, args ...interface{}) {
	fmt.Fprintf(&b.buf, format, args...)
}

func (b *Builder) Expand(tpl string, args map[string]string) {
	s := os.Expand(tpl, func(name string) string {
		return args[name]
	})
	b.Line(s)
}

func (b *Builder) Execute(tpl string, data interface{}) {
	t := template.Must(template.New("").Parse(tpl))
	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		panic(err)
	}
	buf.WriteTo(&b.buf)
}

func (b *Builder) Align(pairs data.Options, indent string, width int) {
	for _, p := range pairs {
		b.Line(indent, texts.PadRight(p.Name, ' ', width), " = ", p.Value)
	}
}

func (b *Builder) Comment(s, prefix string) {
	text := strings.TrimSpace(s)
	if text == "" {
		return
	}

	split := strings.Split(text, "\n")
	for _, line := range split {
		b.Line(prefix, "// ", strings.TrimSpace(line))
	}
}
