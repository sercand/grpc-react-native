package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	gen "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/generator"
)

func ToJsonName(pre string) string {
	if len(pre) == 0 {
		return ""
	}
	word := pre[:1]
	ss := make([]string, 0)
	for i := 1; i < len(pre); i++ {
		letter := pre[i : i+1]
		if word != "" && strings.ToUpper(letter) == letter {
			ss = append(ss, word)
			if letter != "_" && letter != "-" {
				word = letter
			} else {
				word = ""
			}
		} else {
			word += letter
		}
	}
	ss = append(ss, word)
	for i, v := range ss {
		if i != 0 {
			ss[i] = strings.Title(v)
		} else {
			ss[0] = strings.ToLower(ss[0])
		}
	}
	return strings.Join(ss, "")
}

func ToParamName(pre string) string {
	ss := strings.Split(pre, ".")
	return ToJsonName(ss[len(ss)-1])
}

func ToFileName(pre string) string {
	return strings.Title(pre)
}

type generator struct {
	reg       *descriptor.Registry
	mapValues []string
}

// New returns a new generator which generates grpc gateway files.
func NewGenerator(reg *descriptor.Registry) gen.Generator {
	return &generator{reg: reg, mapValues: []string{}}
}
func (g *generator) generate(file *descriptor.File) (string, error) {
	return "", nil
}

func (g *generator) Generate(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var files []*plugin.CodeGeneratorResponse_File
	for _, file := range targets {
		str, err := g.generate(file)
		if err != nil {
			return nil, err
		}
		name := file.GetName()
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		output := fmt.Sprintf("%s.java", ToFileName(base))
		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(str),
		})
		glog.V(1).Infof("Will emit %s", output)
	}

	return files, nil
}
