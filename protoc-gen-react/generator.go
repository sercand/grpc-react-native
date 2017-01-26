package main

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	gdescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
	gen "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/generator"
	"github.com/valyala/fasttemplate"
	"io"
	"path/filepath"
	"strings"
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

func (g *generator) getJavaType(f *gdescriptor.FieldDescriptorProto, file *descriptor.File) string {
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
		m, _ := g.reg.LookupMsg(file.GetPackage(), f.GetTypeName())
		return m.GetName()
	case gdescriptor.FieldDescriptorProto_TYPE_ENUM:
		m, _ := g.reg.LookupMsg(file.GetPackage(), f.GetTypeName())
		return m.GetName()
	}
	return ""
}

func getReactMapType(f *gdescriptor.FieldDescriptorProto) string {
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
func (g *generator) isMap(field *descriptor.Field, file *descriptor.File) bool {
	m, err := g.reg.LookupMsg(file.GetPackage(), field.GetTypeName())
	if err != nil {
		return false
	}
	return m.GetOptions().GetMapEntry()
}

func (g *generator) arrayToBuilder(f *descriptor.Field, file *descriptor.File, mapName string, builderName string, buf io.Writer) error {
	mapType := getReactMapType(f.FieldDescriptorProto)
	if mapType == "Bytes" {
		temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array_{{jsonName}} = {{mapName}}.getArray("{{jsonName}}");
			List<ByteString> list_{{jsonName}} = new ArrayList<>();
			for(int i_{{jsonName}} = 0; i_{{jsonName}} < array_{{jsonName}}.size(); i_{{jsonName}}++){
				list_{{jsonName}}.add(ByteString.copyFromUtf8(array_{{jsonName}}.getString(i_{{jsonName}})));
			}
            {{builderName}}.addAll{{javaName}}(list_{{jsonName}});
        }
`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    g.getJavaType(f.FieldDescriptorProto, file),
			"javaMapType": mapType,
			"builderName": builderName,
		})
	} else if mapType == "Enum" {
		temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array_{{jsonName}} = {{mapName}}.getArray("{{jsonName}}");
			List<{{javaType}}> list_{{jsonName}} = new ArrayList<>();
			for(int i_{{jsonName}} = 0; i_{{jsonName}} < array_{{jsonName}}.size(); i_{{jsonName}}++){
			//	list_{{jsonName}}.add(ByteString.copyFromUtf8(array_{{jsonName}}.getString(i)));
			//	{{builderName}}.set{{javaName}}Value({{mapName}}.getInt("{{jsonName}}"));
			}
            {{builderName}}.addAll{{javaName}}(list_{{jsonName}});
        }
`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    g.getJavaType(f.FieldDescriptorProto, file),
			"javaMapType": mapType,
			"builderName": builderName,
		})
	} else if mapType == "Message" {
		tempStart := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
			ReadableArray array_{{jsonName}} = {{mapName}}.getArray("{{jsonName}}");
			List<{{javaType}}> list_{{jsonName}} = new ArrayList<>();
			for(int i_{{jsonName}} = 0; i_{{jsonName}} < array_{{jsonName}}.size(); i_{{jsonName}}++){
				{{javaType}}.Builder {{jsonName}}_builder = {{javaType}}.newBuilder();
				ReadableMap {{jsonName}}_map = array_{{jsonName}}.getMap(i_{{jsonName}});
`
		tempEnd := `	list_{{jsonName}}.add({{jsonName}}_builder.build());
			}
            {{builderName}}.addAll{{javaName}}(list_{{jsonName}});
        }`
		m, err := g.reg.LookupMsg(file.GetPackage(), f.GetTypeName())
		if err != nil {
			return err
		}
		javaType := g.getJavaType(f.FieldDescriptorProto, file)
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
			ReadableArray array_{{jsonName}} = {{mapName}}.getArray("{{jsonName}}");
			List<{{javaType}}> list_{{jsonName}} = new ArrayList<>();
			for(int i_{{jsonName}} = 0; i_{{jsonName}} < array_{{jsonName}}.size(); i_{{jsonName}}++){
				list_{{jsonName}}.add(array_{{jsonName}}.get{{javaMapType}}(i_{{jsonName}}));
			}
            {{builderName}}.addAll{{javaName}}(list_{{jsonName}});
        }
`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"javaType":    g.getJavaType(f.FieldDescriptorProto, file),
			"javaMapType": mapType,
			"builderName": builderName,
		})
	}
	return nil
}
func (g *generator) mapToBuilder(f *descriptor.Field, file *descriptor.File, mapName string, builderName string, buf io.Writer) error {
	mapEntry, err := g.reg.LookupMsg(file.GetPackage(), f.FieldDescriptorProto.GetTypeName())
	if err != nil {
		return err
	}
	var valueField *gdescriptor.FieldDescriptorProto
	for _, ff := range mapEntry.GetField() {
		if ff.GetName() == "value" {
			valueField = ff
			break
		}
	}
	mapType := getReactMapType(valueField)
	if mapType == "Bytes" {
		temp := `ReadableMap map_{{jsonName}} = {{mapName}}.getMap("{{jsonName}}");
		ReadableMapKeySetIterator iter_{{jsonName}}= map_{{jsonName}}.keySetIterator();
        while(iter_{{jsonName}}.hasNextKey()){
			String key_{{jsonName}}=iter_{{jsonName}}.nextKey();
			String value_{{jsonName}} = map_{{jsonName}}.getString(key_{{jsonName}});
            {{builderName}}.put{{javaName}}(key_{{jsonName}}, ByteString.copyFromUtf8(value_{{jsonName}}));
        }`
		_, err := fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"builderName": builderName,
			"mapType":     mapType,
		})
		return err
	} else if mapType == "Enum" {
		temp := ` ReadableMapKeySetIterator iter_{{jsonName}}={{mapName}}.getMap("{{jsonName}}").keySetIterator();
        while(iter_{{jsonName}}.hasNextKey()){
			String key_{{jsonName}}=iter_{{jsonName}}.nextKey();
            {{builderName}}.put{{javaName}}(key_{{jsonName}}, {{mapName}}.getMap("{{jsonName}}").getInt(key_{{jsonName}}));
        }`
		_, err := fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"builderName": builderName,
			"mapType":     mapType,
		})
		return err
	} else if mapType == "Message" {
		m, err := g.reg.LookupMsg(file.GetPackage(), valueField.GetName())
		if err != nil {
			return err
		}
		javaType := g.getJavaType(valueField, file)
		tempStart := `ReadableMap map_{{jsonName}} = {{mapName}}.getMap("{{jsonName}}");
		ReadableMapKeySetIterator iter_{{jsonName}}= map_{{jsonName}}.keySetIterator();
        while(iter_{{jsonName}}.hasNextKey()){
            {{javaType}}.Builder builder_{{jsonName}} = {{javaType}}.newBuilder();
			String key_{{jsonName}}=iter_{{jsonName}}.nextKey();
			ReadableMap map_{{jsonName}}_inner = map_{{jsonName}}.getMap(key_{{jsonName}});
        `
		tempEnd := `{{builderName}}.put{{javaName}}(key_{{jsonName}}, builder_{{jsonName}}.build());
		}`

		if _, err := fasttemplate.Execute(tempStart, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"builderName": builderName,
			"mapType":     mapType,
			"javaType":    javaType,
		}); err != nil {
			return err
		}
		g.readableMapToBuilder(m, file, fmt.Sprintf("map_%s_inner", f.GetJsonName()), fmt.Sprintf("builder_%s", f.GetJsonName()), buf)

		if _, err := fasttemplate.Execute(tempEnd, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"builderName": builderName,
			"mapType":     mapType,
			"javaType":    javaType,
		}); err != nil {
			return err
		}

	} else {
		temp := `ReadableMap map_{{jsonName}} = {{mapName}}.getMap("{{jsonName}}");
		ReadableMapKeySetIterator iter_{{jsonName}}= map_{{jsonName}}.keySetIterator();
        while(iter_{{jsonName}}.hasNextKey()){
			String key_{{jsonName}}=iter_{{jsonName}}.nextKey();
            {{builderName}}.put{{javaName}}(key_{{jsonName}}, map_{{jsonName}}.get{{mapType}}(key_{{jsonName}}));
        }`

		_, err := fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"builderName": builderName,
			"mapType":     mapType,
		})
		return err
	}
	return nil
}
func (g *generator) readableMapToBuilder(mes *descriptor.Message, file *descriptor.File, mapName string, builderName string, buf io.Writer) error {
	for _, f := range mes.Fields {
		javaName := f.GetJsonName()
		mapType := getReactMapType(f.FieldDescriptorProto)
		isArray := f.GetLabel() == gdescriptor.FieldDescriptorProto_LABEL_REPEATED

		if g.isMap(f, file) {
			temp := `if ({{mapName}}.hasKey("{{jsonName}}")) {
`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName": f.GetJsonName(),
				"mapName":  mapName,
			})
			if err := g.mapToBuilder(f, file, mapName, builderName, buf); err != nil {
				return err
			}
			buf.Write([]byte("}"))
		} else if isArray {
			if err := g.arrayToBuilder(f, file, mapName, builderName, buf); err != nil {
				return err
			}
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
			javaType := g.getJavaType(f.FieldDescriptorProto, file)
			temp := `		if ({{mapName}}.hasKey("{{jsonName}}")) {
				{{javaType}}.Builder builder_{{jsonName}} = {{javaType}}.newBuilder();
				ReadableMap in_{{jsonName}} = {{mapName}}.getMap("{{jsonName}}");
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

			tempEnd := `{{builderName}}.set{{javaName}}(builder_{{jsonName}});
				}`
			fasttemplate.Execute(tempEnd, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(javaName),
				"mapType":     mapType,
				"mapName":     mapName,
				"builderName": builderName,
				"javaType":    javaType,
			})
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
	return nil
}

func (g *generator) javaMapToMap(f *descriptor.Field, file *descriptor.File, mapName string, messageName string, buf io.Writer) error {
	innerMap := fmt.Sprintf("%s_%s", mapName, f.GetJsonName())
	javaMap := fmt.Sprintf("%s_%s", messageName, f.GetJsonName())
	mapEntry, err := g.reg.LookupMsg(file.GetPackage(), f.FieldDescriptorProto.GetTypeName())
	if err != nil {
		return err
	}
	var valueField *gdescriptor.FieldDescriptorProto
	for _, ff := range mapEntry.GetField() {
		if ff.GetName() == "value" {
			valueField = ff
			break
		}
	}
	mapType := getReactMapType(valueField)
	tempStart := `
			Map<String,{{JavaValue}}> {{javaMap}} = {{messageName}}.get{{javaName}}Map();
			WritableMap {{innerMap}} = Arguments.createMap();
			for(String k_{{javaMap}}: {{javaMap}}.keySet()){
				{{JavaValue}} v_{{javaMap}} = {{javaMap}}.get(k_{{javaMap}});
			`
	javaType := g.getJavaType(valueField, file)

	fasttemplate.Execute(tempStart, "{{", "}}", buf, map[string]interface{}{
		"jsonName":    f.GetJsonName(),
		"javaName":    strings.Title(f.GetJsonName()),
		"mapName":     mapName,
		"messageName": messageName,
		"innerMap":    innerMap,
		"javaMap":     javaMap,
		"JavaValue":   javaType,
	})
	tempEnd := `}
	{{mapName}}.putMap("{{jsonName}}",{{innerMap}});`

	if mapType == "Enum" {
		tempEnum := `{{innerMap}}.putInt(k_{{javaMap}},v_{{javaMap}}.getNumber());
		`
		fasttemplate.Execute(tempEnum, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"messageName": messageName,
			"innerMap":    innerMap,
			"javaMap":     javaMap,
			"JavaValue":   javaType,
		})
	} else if mapType == "Bytes" {
		tempBytes := `{{innerMap}}.putString(k_{{javaMap}},v_{{javaMap}}.toStringUtf8());
		`
		fasttemplate.Execute(tempBytes, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"messageName": messageName,
			"innerMap":    innerMap,
			"javaMap":     javaMap,
			"JavaValue":   javaType,
		})
	} else if mapType == "Message" {
		javaType := g.getJavaType(f.FieldDescriptorProto, file)
		mesfield, _ := g.reg.LookupMsg(file.GetPackage(), valueField.GetTypeName())
		tempStart := `WritableMap in_{{innerMap}}_map = Arguments.createMap();`
		tempEnd := `{{innerMap}}.putMap(k_{{javaMap}},in_{{innerMap}}_map);
			`
		fasttemplate.Execute(tempStart, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"messageName": messageName,
			"javaType":    javaType,
			"innerMap":    innerMap,
			"javaMap":     javaMap,
		})
		g.messageToMap(mesfield, file, fmt.Sprintf("in_%s_map", innerMap), fmt.Sprintf("v_%s", javaMap), buf)
		fasttemplate.Execute(tempEnd, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"messageName": messageName,
			"innerMap":    innerMap,
			"javaMap":     javaMap,
		})
	} else {
		temp := `{{innerMap}}.put{{mapType}}(k_{{javaMap}},v_{{javaMap}});
		`
		fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
			"jsonName":    f.GetJsonName(),
			"javaName":    strings.Title(f.GetJsonName()),
			"mapName":     mapName,
			"messageName": messageName,
			"innerMap":    innerMap,
			"javaMap":     javaMap,
			"JavaValue":   javaType,
			"mapType":     mapType,
		})
	}

	fasttemplate.Execute(tempEnd, "{{", "}}", buf, map[string]interface{}{
		"jsonName": f.GetJsonName(),
		"mapName":  mapName,
		"innerMap": innerMap,
	})

	return nil
}

func (g *generator) arrayToMap(f *descriptor.Field, file *descriptor.File, mapName string, messageName string, buf io.Writer) error {
	return nil
}

func (g *generator) messageToMap(mes *descriptor.Message, file *descriptor.File, mapName string, messageName string, buf io.Writer) error {
	for _, f := range mes.Fields {
		mapType := getReactMapType(f.FieldDescriptorProto)
		isArray := f.GetLabel() == gdescriptor.FieldDescriptorProto_LABEL_REPEATED

		if g.isMap(f, file) {
			g.javaMapToMap(f, file, mapName, messageName, buf)
		} else if isArray {
			g.arrayToMap(f, file, mapName, messageName, buf)
		} else if mapType == "Enum" {
			temp := `{{mapName}}.putInt("{{jsonName}}",{{messageName}}.get{{javaName}}Value());
		`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(f.GetJsonName()),
				"mapName":     mapName,
				"messageName": messageName,
			})
		} else if mapType == "Bytes" {
			temp := `{{mapName}}.putString("{{jsonName}}",{{messageName}}.get{{javaName}}().toStringUtf8());
		`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(f.GetJsonName()),
				"mapName":     mapName,
				"messageName": messageName,
			})
		} else if mapType == "Message" {
			javaType := g.getJavaType(f.FieldDescriptorProto, file)
			mesfield, _ := g.reg.LookupMsg(file.GetPackage(), f.GetTypeName())
			tempStart := `
			WritableMap {{mapName}}_{{jsonName}} = Arguments.createMap();
			{{javaType}} {{messageName}}_{{jsonName}} = {{messageName}}.get{{javaName}}();
			`
			tempEnd := `{{mapName}}.putMap("{{jsonName}}",{{mapName}}_{{jsonName}});

			`
			fasttemplate.Execute(tempStart, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(f.GetJsonName()),
				"mapName":     mapName,
				"messageName": messageName,
				"javaType":    javaType,
			})
			g.messageToMap(mesfield, file, fmt.Sprintf("%s_%s", mapName, f.GetJsonName()), fmt.Sprintf("%s_%s", messageName, f.GetJsonName()), buf)
			fasttemplate.Execute(tempEnd, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(f.GetJsonName()),
				"mapName":     mapName,
				"messageName": messageName,
			})
		} else if mapType == "Int" {
			//TODO this is sucks use string for long values
			temp := `{{mapName}}.putInt("{{jsonName}}",(int){{messageName}}.get{{javaName}}());
		`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(f.GetJsonName()),
				"mapName":     mapName,
				"messageName": messageName,
			})
		} else {
			temp := `{{mapName}}.put{{mapType}}("{{jsonName}}",{{messageName}}.get{{javaName}}());
		`
			fasttemplate.Execute(temp, "{{", "}}", buf, map[string]interface{}{
				"jsonName":    f.GetJsonName(),
				"javaName":    strings.Title(f.GetJsonName()),
				"mapType":     mapType,
				"mapName":     mapName,
				"messageName": messageName,
			})
		}
	}
	return nil
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
            	if(result == null){
            		promise.reject("null","response is null");
            		return;
            	}
                WritableMap out = Arguments.createMap();
                `
	fasttemplate.Execute(futureTemp, "{{", "}}", buf, map[string]interface{}{
		"responseName": m.ResponseType.GetName(),
		"methodName":   name,
	})

	if err := g.messageToMap(m.ResponseType, file, "out", "result", buf); err != nil {
		return err
	}

	_, err := buf.Write([]byte(`promise.resolve(out);
            }

            @Override
            public void onFailure(Throwable t) {
                promise.reject(t);
            }
        });
	}`))
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
