// The following directive is necessary to make the package coherent:
//go:build ignore

// This program generates contributors.go. It can be invoked by running
// go generate
package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	TAB = "\t"
)

var imports = map[string]string{
	"uuid": "github.com/google/uuid",
}

var simpleName = map[string]string{
	"byte[]":   "ByteArray",
	"string[]": "StringArray",
	"uuid":     "Uuid",
}

var typeMap = map[string]string{
	"byte":     "byte",
	"byte[]":   "*[]byte",
	"short":    "int16",
	"int":      "int32",
	"long":     "int64",
	"string":   "*string",
	"string[]": "*[]string",
	"uuid":     "*uuid.UUID",
}

type Field struct {
	Name      string
	Type      string
	KeyType   string
	ValueType string
	Value     string
	Unsafe    bool
}

type Request struct {
	Fields   []Field
	Optional []Field
}

type Format struct {
	Type         string
	Name         string
	Code         int16
	Request      *Request
	Response     *Request
	FailResponse *Request
}

func main() {
	serdesDir := "./serdes"

	err := os.MkdirAll(serdesDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	generateMapsSerdes(serdesDir)
	generateFormats(serdesDir)
}

func generateMapsSerdes(serdesDir string) {
	fmt.Println("Generating maps serdes methods")

	f, err := os.Create(filepath.Join(serdesDir, "gen_maps_serdes.go"))
	defer f.Close()
	if err != nil {
		panic(err)
	}

	writeHeader(f)
	line("import \"github.com/google/uuid\"", f)
	line("", f)

	types := make([]string, len(typeMap))
	i := 0
	for k := range typeMap {
		types[i] = k
		i++
	}
	sort.Strings(types)

	for i, keyType := range types {
		goKeyType := typeMap[keyType]
		if strings.HasSuffix(keyType, "[]") {
			fmt.Println(fmt.Sprintf("Skip map key [key=%s]", keyType))
			continue
		}

		isGoKeyTypePointer := goKeyType[0] == '*'
		goKeyType = removePointer(goKeyType)

		for j, valType := range types {
			if j != 0 || i != 0 {
				line("", f)
			}
			goValType := typeMap[valType]
			isGoValTypePointer := goValType[0] == '*'
			goValType = removePointer(goValType)

			line(fmt.Sprintf("func (buf *IgniteBuffer) %s(m *map[%s]%s) {", mapWriteMethodName(keyType, valType), goKeyType, goValType), f)
			line(TAB+"if m == nil {", f)
			line(TAB+TAB+"buf.WriteByte(Null)", f)
			line(TAB+TAB+"return", f)
			line(TAB+"}", f)

			line(TAB+"buf.WriteByte(MAP)", f)
			line(TAB+"buf.WriteInt(len(*m))", f)
			line(TAB+"buf.WriteByte(1)", f)
			line("", f)
			line(TAB+"for key, val := range *m {", f)

			keyArg := "key"
			if isGoKeyTypePointer {
				keyArg = "&key"
			}

			valArg := "val"
			if isGoValTypePointer {
				valArg = "&val"
			}

			line(TAB+TAB+fmt.Sprintf("buf.%s(%s)", writeMethodName(keyType, nil), keyArg), f)

			line(TAB+TAB+fmt.Sprintf("buf.%s(%s)", writeMethodName(valType, nil), valArg), f)
			line(TAB+"}", f)

			line("}", f)
		}
	}
}

func generateFormats(serdesDir string) {
	fmt.Println("Generating Ignite Thin Client request/response structs")
	files, err := filepath.Glob("../formats/*.json")
	if err != nil {
		panic(err)
	}

	for _, formatFile := range files {
		generateFrom(formatFile, serdesDir)
	}
}

func generateFrom(formatFile string, serdesDir string) {
	fmt.Printf("Process file [file=%s]", formatFile)
	fmt.Println()
	formatBytes, _ := os.ReadFile(formatFile)

	var desc Format

	err := json.Unmarshal(formatBytes, &desc)
	if err != nil {
		panic(err)
	}

	fmt.Printf("|--> request [name=%s]", desc.Name)
	fmt.Println()

	generateRequest(&desc, serdesDir)
}

func generateRequest(format *Format, serdesDir string) {
	f, err := os.Create(filepath.Join(serdesDir, "gen_"+format.Name+".go"))
	defer f.Close()
	if err != nil {
		panic(err)
	}

	writeHeader(f)
	if generateImports(format.Request, f) || generateImports(format.Response, f) {
		line("", f)
	}

	requestType := exportedName(camelCase(format.Name)) + "Request"
	responseType := exportedName(camelCase(format.Name)) + "Response"

	genStruct(format.Request, requestType, f)
	line("", f)
	genStruct(format.Response, responseType, f)
	line("", f)
	genOpCode(format.Request, requestType, format.Code, f)
	line("", f)
	genConstructor(format.Request, requestType, f)
	line("", f)
	genWrite(format.Request, requestType, f)
	line("", f)
	genReadResponse(format.Response, requestType, responseType, "ReadResponse", f)

	if format.FailResponse != nil {
		failResponseType := exportedName(camelCase(format.Name)) + "FailResponse"
		line("", f)
		genStruct(format.FailResponse, failResponseType, f)
		line("", f)
		genReadResponse(format.FailResponse, requestType, failResponseType, "ReadFailResponse", f)
	}
}

func generateImports(request *Request, f *os.File) bool {
	res := false

	addImport := func(fld Field) bool {
		imprt, contains := imports[fld.Type]
		if contains {
			line(fmt.Sprintf("import \"%s\"", imprt), f)
		}
		return contains
	}

	for _, fld := range request.Fields {
		res = res || addImport(fld)
	}

	for _, fld := range request.Optional {
		res = res || addImport(fld)
	}

	return res
}

func genStruct(request *Request, typeName string, f *os.File) {
	line(fmt.Sprintf("type %s struct {", typeName), f)

	maxNameLen := 0
	for _, fld := range request.Fields {
		if maxNameLen < len(fld.Name) {
			maxNameLen = len(fld.Name)
		}
	}
	for _, fld := range request.Optional {
		if maxNameLen < len(fld.Name) {
			maxNameLen = len(fld.Name)
		}
	}

	fldFormat := TAB + "%-" + strconv.Itoa(maxNameLen) + "s %s"

	for _, fld := range request.Fields {
		line(fmt.Sprintf(fldFormat, exportedName(fld.Name), goType(fld.Type, &fld)), f)
	}
	for _, fld := range request.Optional {
		line(fmt.Sprintf(fldFormat, exportedName(fld.Name), goType(fld.Type, &fld)), f)
	}

	line("}", f)
}

func genOpCode(request *Request, typeName string, opCode int16, f *os.File) {
	line(fmt.Sprintf("func (req %s) OpCode() int16 {", typeName), f)
	line(fmt.Sprintf(TAB+"return %d", opCode), f)
	line("}", f)
}

func genReadResponse(response *Request, requestType string, responseType string, mName string, f *os.File) {
	line(fmt.Sprintf("func (req %s) %s(buf *IgniteBuffer) interface{} {", requestType, mName), f)
	line(fmt.Sprintf(TAB+"resp := %s{}", responseType), f)

	for _, fld := range response.Fields {
		line(fmt.Sprintf(TAB+"resp.%s = buf.%s()", exportedName(fld.Name), readMethodName(fld)), f)
	}

	if len(response.Optional) > 0 {
		line("", f)
		line(TAB+"if buf.HasMore() {", f)

		for _, fld := range response.Optional {
			line(fmt.Sprintf(TAB+TAB+"resp.%s = buf.%s()", exportedName(fld.Name), readMethodName(fld)), f)
		}

		line(TAB+"}", f)
	}

	line(TAB+"return resp", f)
	line("}", f)
}

func genConstructor(request *Request, typeName string, f *os.File) {
	line(fmt.Sprintf("func %s() %s {", constructorName(typeName), typeName), f)

	var dfltVals string

	for _, fld := range request.Fields {
		if fld.Value != "" {
			if dfltVals != "" {
				dfltVals += ", "
			}
			dfltVals += fmt.Sprintf("%s: %s", exportedName(fld.Name), defaultValue(fld))
		}
	}

	line(fmt.Sprintf(TAB+"return %s{%s}", typeName, dfltVals), f)
	line("}", f)
}

func genWrite(request *Request, typeName string, f *os.File) {
	line(fmt.Sprintf("func (req %s) Write(buf *IgniteBuffer) {", typeName), f)

	for _, fld := range request.Fields {
		line(fmt.Sprintf(TAB+"buf.%s(req.%s)", writeMethodName(fld.Type, &fld), exportedName(fld.Name)), f)
	}

	line("}", f)
}

func writeHeader(f *os.File) {
	line("// Code generated by \"gen_req_resp.go\"; DO NOT EDIT.", f)
	line("package serdes", f)
	line("", f)
}

func line(line string, file *os.File) {
	file.WriteString(line)
	file.WriteString("\n")
}

func writeMethodName(tp string, fld *Field) string {
	if fld != nil && fld.Type == "map" {
		keyTypeName, contains := simpleName[fld.KeyType]
		if !contains {
			keyTypeName = fld.KeyType
		}

		valTypeName, contains0 := simpleName[fld.ValueType]
		if !contains0 {
			valTypeName = fld.ValueType
		}

		return mapWriteMethodName(keyTypeName, valTypeName)
	}

	return methodName("Write", tp)
}

func mapWriteMethodName(keyType string, valType string) string {
	keyPart, contains := simpleName[keyType]
	if !contains {
		keyPart = exportedName(removePointer(goType(keyType, nil)))
	}
	valPart, contains := simpleName[valType]
	if !contains {
		valPart = exportedName(removePointer(goType(valType, nil)))
	}

	return fmt.Sprintf("Write%s%s", keyPart, valPart)
}

func readMethodName(fld Field) string {
	res := methodName("Read", fld.Type)
	if fld.Unsafe {
		return res + "WithoutType"
	}
	return res
}

func methodName(pfx string, tp string) string {
	name, contains := simpleName[tp]
	if !contains {
		goTp := removePointer(goType(tp, nil))

		return fmt.Sprintf("%s%s", pfx, exportedName(goTp))
	}
	return pfx + name
}

func constructorName(typeName string) string {
	return fmt.Sprintf("Create%s", typeName)
}

func exportedName(name string) string {
	return cases.Title(language.English, cases.NoLower).String(name)
}

func camelCase(name string) string {
	parts := strings.Split(name, "_")
	res := ""
	for _, part := range parts {
		res += strings.ToUpper(part[:1]) + part[1:]
	}

	return res
}

func goType(tp string, fld *Field) string {
	if tp == "map" {
		if fld == nil {
			panic("fld is nil")
		}
		return fmt.Sprintf("*map[%s]%s", removePointer(goType(fld.KeyType, nil)), removePointer(goType(fld.ValueType, nil)))
	}

	res, ok := typeMap[tp]
	if !ok {
		panic(fmt.Sprintf("unknown type %s", tp))
	}
	return res
}

func defaultValue(fld Field) string {
	if fld.Type == "string" {
		return fmt.Sprintf("\"%s\"", fld.Value)
	}

	_, ok := typeMap[fld.Type]
	if !ok {
		panic("Unknown type " + fld.Type)
	}

	return fld.Value
}

func removePointer(tp string) string {
	if strings.HasPrefix(tp, "*") {
		return tp[1:]
	}

	return tp
}
