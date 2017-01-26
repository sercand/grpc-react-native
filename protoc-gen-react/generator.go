package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"bytes"
	"io"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	gdescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	gen "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/generator"
	"github.com/valyala/fasttemplate"
)

const (
	javaHeader = `// Code generated by protoc-gen-react
// DO NOT EDIT!
package {{packageName}};

import com.facebook.react.bridge.*;
import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.Futures;
import io.grpc.ManagedChannel;
import com.google.protobuf.ByteString;

import javax.annotation.Nullable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import {{protoPackages}}.*;
`
	serviceTop = `
class {{serviceName}}Module extends ReactContextBaseJavaModule {
    private GrpcEngine engine;

    {{serviceName}}Module(ReactApplicationContext reactContext, GrpcEngine engine) {
        super(reactContext);
        this.engine = engine;
    }

    /**
     * @return the name of this module. This will be the name used to {@code require()} this module
     * from javascript.
     */
    @Override
    public String getName() {
        return "{{serviceName}}";
    }

    @Override
    public Map<String, Object> getConstants() {
        final Map<String, Object> constants = new HashMap<>();
        constants.put("NAME", {{serviceName}}Grpc.SERVICE_NAME);
        return constants;
    }
`
)

func ToJsonName(pre string) string {
	if len(pre) == 0 {
		return ""
	}
	word := pre[:1]
	ss := make([]string, 0)
	for i := 1; i < len(pre); i++ {
		letter := pre[i : i + 1]
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
	return ToJsonName(ss[len(ss) - 1])
}

func ToFileName(pre string) string {
	return strings.Title(pre)
}

type generator struct {
	reg         *descriptor.Registry
	mapValues   []string
	packageName string
}

// New returns a new generator which generates grpc gateway files.
func NewGenerator(reg *descriptor.Registry, packageName string) gen.Generator {
	return &generator{reg: reg, mapValues: []string{}, packageName: packageName}
}

func (g *generator) getJavaType(f *descriptor.Field, file *descriptor.File) string {

	switch f.GetType() {
	case gdescriptor.FieldDescriptorProto_TYPE_BOOL:
		return "boolean"
	case gdescriptor.FieldDescriptorProto_TYPE_STRING:
		return "String"
	case gdescriptor.FieldDescriptorProto_TYPE_INT32,
		gdescriptor.FieldDescriptorProto_TYPE_SFIXED32,
		gdescriptor.FieldDescriptorProto_TYPE_SINT32:
		return "long"
	case gdescriptor.FieldDescriptorProto_TYPE_INT64,
		gdescriptor.FieldDescriptorProto_TYPE_SFIXED64,
		gdescriptor.FieldDescriptorProto_TYPE_SINT64:
		return "int"
	case gdescriptor.FieldDescriptorProto_TYPE_FLOAT:
		return "float"
	case gdescriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return "double"
	case gdescriptor.FieldDescriptorProto_TYPE_MESSAGE:
		m, _ := g.reg.LookupMsg(file.GetPackage(), f.FieldDescriptorProto.GetTypeName())
		return m.GetName()
	case gdescriptor.FieldDescriptorProto_TYPE_ENUM:
		m, _ := g.reg.LookupMsg(file.GetPackage(), f.FieldDescriptorProto.GetTypeName())
		return m.GetName()
	}
	return ""
}

func getReactMapType(f *descriptor.Field) string {
	switch f.GetType() {
	case gdescriptor.FieldDescriptorProto_TYPE_BOOL:
		return "Boolean"
	case gdescriptor.FieldDescriptorProto_TYPE_STRING:
		return "String"
	case gdescriptor.FieldDescriptorProto_TYPE_INT32,
		gdescriptor.FieldDescriptorProto_TYPE_INT64,
		gdescriptor.FieldDescriptorProto_TYPE_SFIXED32,
		gdescriptor.FieldDescriptorProto_TYPE_SFIXED64,
		gdescriptor.FieldDescriptorProto_TYPE_SINT32,
		gdescriptor.FieldDescriptorProto_TYPE_SINT64:
		return "Int"
	case gdescriptor.FieldDescriptorProto_TYPE_FLOAT,
		gdescriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return "Double"
	case gdescriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return "Message"
	case gdescriptor.FieldDescriptorProto_TYPE_ENUM:
		return "Enum"
	case gdescriptor.FieldDescriptorProto_TYPE_BYTES:
		return "Bytes"
	}
	return ""
}

func (g *generator) arrayToBuilder(f *descriptor.Field, file *descriptor.File, mapName string, builderName string, buf io.Writer) error {
	mapType := getReactMapType(f)
	if mapType == "Bytes" {
		temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array = {{mapName}}.getArray("{{jsonName}}");
			List<ByteString> list = new ArrayList<>();
			for(int i = 0; i < array.size(); i++){
				list.add(ByteString.copyFromUtf8(array.getString(i)));
			}
            {{builderName}}.addAll{{javaName}}(list);
        }
`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    g.getJavaType(f, file),
			"javaMapType": mapType,
			"builderName": builderName,
		})
	} else if mapType == "Enum" {
		temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array = {{mapName}}.getArray("{{jsonName}}");
			List<{{javaType}}> list = new ArrayList<>();
			for(int i = 0; i < array.size(); i++){
			//	list.add(ByteString.copyFromUtf8(array.getString(i)));
			//	{{builderName}}.set{{javaName}}Value({{mapName}}.getInt("{{jsonName}}"));
			}
            {{builderName}}.addAll{{javaName}}(list);
        }
`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    g.getJavaType(f, file),
			"javaMapType": mapType,
			"builderName": builderName,
		})
	} else if mapType == "Message" {
		tempStart := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array = {{mapName}}.getArray("{{jsonName}}");
			List<{{javaType}}> list = new ArrayList<>();
			for(int i = 0; i < array.size(); i++){
				{{javaType}}.Builder {{jsonName}}_builder = {{javaType}}.newBuilder();
				ReadableMap {{jsonName}}_map = array.getMap(i);
`
		tempEnd := `	list.add({{jsonName}}_builder.build());
			}
            {{builderName}}.addAll{{javaName}}(list);
        }`
		m, err := g.reg.LookupMsg(file.GetPackage(), f.FieldDescriptorProto.GetTypeName())
		if err != nil {
			return err
		}
		javaType := g.getJavaType(f, file)
		fasttemplate.Execute(tempStart, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    javaType,
			"javaMapType": mapType,
			"builderName": builderName,
		})

		g.readableMapToBuilder(m, file, fmt.Sprintf("%s_map", f.GetJsonName()), fmt.Sprintf("%s_builder", f.GetJsonName()), buf)
		fasttemplate.Execute(tempEnd, "{{", "}}", buf, map[string]interface{}{
			"javaName":    strings.Title(f.GetJsonName()),
			"builderName": builderName,
			"jsonName":    f.GetJsonName(),
		})
	} else {
		temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array = {{mapName}}.getArray("{{jsonName}}");
			List<{{javaType}}> list = new ArrayList<>();
			for(int i = 0; i < array.size(); i++){
				list.add(array.get{{javaMapType}}(i));
			}
            {{builderName}}.addAll{{javaName}}(list);
        }
`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    g.getJavaType(f, file),
			"javaMapType": mapType,
			"builderName": builderName,
		})
	}
	return nil
}

func (g *generator) readableMapToBuilder(mes *descriptor.Message, file *descriptor.File, mapName string, builderName string, buf io.Writer) {
	for _, f := range mes.Fields {
		javaName := f.GetJsonName()
		mapType := getReactMapType(f)
		isArray := f.GetLabel() == gdescriptor.FieldDescriptorProto_LABEL_REPEATED
		if isArray {
			g.arrayToBuilder(f, file, mapName, builderName, buf)
		} else if mapType == "Enum" {
			temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
            {{builderName}}.set{{javaName}}Value({{mapName}}.getInt("{{jsonName}}"));
        }
`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(javaName),
				"mapName":     mapName,
				"builderName": builderName,
			})
		} else if mapType == "Message" {
			javaType := g.getJavaType(f, file)
			temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
				{{javaType}}.Builder builder_{{jsonName}} = {{javaType}}.newBuilder();
				ReadableMap in_{{jsonName}} = {{mapName}}.getMap("{{jsonName}}")
`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(javaName),
				"mapType":     mapType,
				"mapName":     mapName,
				"builderName": builderName,
				"javaType":    javaType,
			})
			mes, _ := g.reg.LookupMsg(file.GetPackage(), f.GetTypeName())
			g.readableMapToBuilder(mes, file,
				fmt.Sprintf("in_%s", f.GetJsonName()),
				fmt.Sprintf("builder_%s", f.GetJsonName()), buf)
			buf.Write([]byte("}"))
		} else if mapType == "Bytes" {
			//TODO use base64
			temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
            {{builderName}}.set{{javaName}}(ByteString.copyFromUtf8({{mapName}}.getString("{{jsonName}}")));
        }
`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(javaName),
				"mapType":     mapType,
				"mapName":     mapName,
				"builderName": builderName,
			})
		} else {
			temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
            {{builderName}}.set{{javaName}}({{mapName}}.get{{mapType}}("{{jsonName}}"));
        }
`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(javaName),
				"mapType":     mapType,
				"mapName":     mapName,
				"builderName": builderName,
			})
		}
	}
}
func (g *generator) messageToMap(mes *descriptor.Message, file *descriptor.File, mapName string, messageName string, buf io.Writer) {

}

func (g *generator) generateUnaryMethod(m *descriptor.Method, file *descriptor.File, buf io.Writer) error {
	name := ToJsonName(m.GetName())
	met := `
	@ReactMethod
    public void {{methodName}}(ReadableMap in, final Promise promise) {
        ManagedChannel ch = this.engine.byServiceName({{serviceName}}Grpc.SERVICE_NAME);
        {{serviceName}}Grpc.{{serviceName}}FutureStub stub = {{serviceName}}Grpc.newFutureStub(ch);
        {{requestName}}.Builder builder = {{requestName}}.newBuilder();
`

	fasttemplate.Execute(met, "{{", "}}", buf, map[string]interface{}{
		"serviceName": m.Service.GetName(),
		"methodName":  name,
		"requestName": m.RequestType.GetName(),
	})

	g.readableMapToBuilder(m.RequestType, file, "in", "builder", buf)
	futureTemp := `
        Futures.addCallback(stub.{{methodName}}(builder.build()), new FutureCallback<{{responseName}}>() {
            @Override
            public void onSuccess(@Nullable {{responseName}} result) {
                WritableMap out = Arguments.createMap();
                `
	fasttemplate.Execute(futureTemp, "{{", "}}", buf, map[string]interface{}{
		"responseName": m.ResponseType.GetName(),
		"methodName":   name,
	})

	g.messageToMap(m.ResponseType, file, "out", "result", buf)

	buf.Write([]byte(`promise.resolve(out);
            }

            @Override
            public void onFailure(Throwable t) {
                promise.reject(t);
            }
        });`))

	_, err := buf.Write([]byte("\n    }\n"))

	return err
}

func (g *generator) generateMethod(m *descriptor.Method, file *descriptor.File, buf io.Writer) error {
	if m.GetClientStreaming() == false && m.GetServerStreaming() == false {
		return g.generateUnaryMethod(m, file, buf)
	}
	return nil
}

func (g *generator) generate(file *descriptor.File) (string, error) {
	var buf bytes.Buffer
	var pn = g.packageName
	if pn == "" {
		pn = file.Options.GetJavaPackage()
	}
	fasttemplate.Execute(javaHeader, "{{", "}}", &buf, map[string]interface{}{
		"packageName":   pn,
		"protoPackages": file.Options.GetJavaPackage(),
	})

	for _, svc := range file.Services {
		glog.V(1).Infof("Service Name %s", svc.GetName())

		fasttemplate.Execute(serviceTop, "{{", "}}", &buf, map[string]interface{}{
			"serviceName": svc.GetName(),
		})

		for _, m := range svc.Methods {
			if err := g.generateMethod(m, file, &buf); err != nil {
				return "", err
			}
		}

		buf.WriteString("}")
	}

	return buf.String(), nil
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
		output := fmt.Sprintf("%sModule.java", ToFileName(base))
		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(str),
		})
		glog.V(1).Infof("Will emit %s", output)
	}

	return files, nil
}
