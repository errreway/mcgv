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
	TAB               = "\t"
	TYPES             = "types.json"
	arraySuffix       = "[]"
	withoutTypeSuffix = "WithoutType"
	read              = "Read"
	write             = "Write"
)

var imports = map[string]string{
	"uuid": "github.com/google/uuid",
}

var simpleName = map[string]string{
	"uuid": "Uuid",
}

var typeMap = map[string]string{
	"byte":     "byte",
	"boolean":  "bool",
	"byte[]":   "*[]byte",
	"short":    "int16",
	"int":      "int32",
	"long":     "int64",
	"string":   "*string",
	"string[]": "*[]string",
	"uuid":     "*uuid.UUID",
}

var customTypes = map[string]string{}

type Field struct {
	Name      string
	Type      string
	KeyType   string
	ValueType string
	Value     string
	Unsafe    bool
}

type Type struct {
	Name     string
	Fields   []Field
	Optional []Field
}

type Format struct {
	Name         string
	Code         int16
	Request      *Type
	Response     *Type
	FailResponse *Type
}

type TypesList struct {
	Types []Type
}

func main() {
	serdesDir := "./serdes"

	err := os.MkdirAll(serdesDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	generateMapsSerdes(serdesDir)
	generateTypes(serdesDir)
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

	types := sortedKeys(typeMap)
	for i, keyType := range types {
		if isArrayType(keyType) {
			fmt.Println(fmt.Sprintf("Skip map key [key=%s]", keyType))
			continue
		}

		for j, valType := range types {
			if j != 0 || i != 0 {
				line("", f)
			}
			generateMapWrite(keyType, valType, f)
			generateMapRead(keyType, valType, f)
		}
	}
}

func generateTypes(serdesDir string) {
	fmt.Println("Generating Ignite Thin Client types structs")

	formatBytes, _ := os.ReadFile("../formats/" + TYPES)

	var typesList TypesList

	err := json.Unmarshal(formatBytes, &typesList)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filepath.Join(serdesDir, "gen_types.go"))
	defer f.Close()
	if err != nil {
		panic(err)
	}

	writeHeader(f)

	arrayWriteMethodRequired := make(map[string]bool)
	typesMap := make(map[string]Type)

	for _, tp := range typesList.Types {
		fmt.Println(fmt.Sprintf("|--> type [name=%s]", tp.Name))
		typesMap[tp.Name] = tp
		generateStruct(&tp, tp.Name, f)
		customTypes[tp.Name] = tp.Name
		customTypes[tp.Name+arraySuffix] = arraySuffix + tp.Name
		line("", f)
	}

	for _, tp := range typesList.Types {
		fmt.Println(fmt.Sprintf("|--> type [name=%s]", tp.Name))
		generateWrite(&tp, tp.Name, f)
		line("", f)

		generateRead(&tp, tp.Name, fmt.Sprintf("func (buf *IgniteBuffer) %s() %s {", read+tp.Name, tp.Name), f)
		line("", f)

		forEachField(&tp, func(fld Field) {
			if isArrayType(fld.Type) {
				componentType := arrayComponentType(fld.Type)
				_, contains := typesMap[componentType]
				if !contains {
					panic(fmt.Sprintf("unknown array type[type=%s]", componentType))
				}

				arrayWriteMethodRequired[componentType] = fld.Unsafe
			}
		})
	}

	for i, compType := range sortedKeys(arrayWriteMethodRequired) {
		unsafe := arrayWriteMethodRequired[compType]
		fmt.Println(fmt.Sprintf("|--> array [name=%s]", compType))
		generateWriteArray(typesMap[compType], unsafe, f)
		line("", f)
		generateReadArray(typesMap[compType], unsafe, f)
		if i+1 < len(arrayWriteMethodRequired) {
			line("", f)
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
		if strings.HasSuffix(formatFile, TYPES) {
			continue
		}
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

	generateStruct(format.Request, requestType, f)
	line("", f)
	generateStruct(format.Response, responseType, f)
	line("", f)
	generateOpCode(format.Request, requestType, format.Code, f)
	line("", f)
	if generateConstructor(format.Request, requestType, f) {
		line("", f)
	}
	generateWrite(format.Request, requestType, f)
	line("", f)
	generateRead(format.Response, responseType, fmt.Sprintf("func (req %s) ReadResponse(buf *IgniteBuffer) interface{} {", requestType), f)

	if format.FailResponse != nil {
		failResponseType := exportedName(camelCase(format.Name)) + "FailResponse"
		line("", f)
		generateStruct(format.FailResponse, failResponseType, f)
		line("", f)
		generateRead(format.FailResponse, failResponseType, fmt.Sprintf("func (req %s) ReadFailResponse(buf *IgniteBuffer) interface{} {", requestType), f)
	}
}

func generateImports(request *Type, f *os.File) bool {
	res := false

	addImport := func(fld Field) bool {
		imprt, contains := imports[fld.Type]
		if contains {
			line(fmt.Sprintf("import \"%s\"", imprt), f)
		}
		return contains
	}

	forEachField(request, func(fld Field) {
		res = res || addImport(fld)
	})

	return res
}

func generateStruct(request *Type, typeName string, f *os.File) {
	line(fmt.Sprintf("type %s struct {", typeName), f)

	maxNameLen := 0
	forEachField(request, func(fld Field) {
		if maxNameLen < len(fld.Name) {
			maxNameLen = len(fld.Name)
		}
	})

	fldFormat := TAB + "%-" + strconv.Itoa(maxNameLen) + "s %s"

	forEachField(request, func(fld Field) {
		line(fmt.Sprintf(fldFormat, exportedName(fld.Name), goType(fld.Type, &fld)), f)
	})

	line("}", f)
}

func generateOpCode(request *Type, typeName string, opCode int16, f *os.File) {
	line(fmt.Sprintf("func (req %s) OpCode() int16 {", typeName), f)
	line(fmt.Sprintf(TAB+"return %d", opCode), f)
	line("}", f)
}

func generateRead(response *Type, responseType string, signature string, f *os.File) {
	line(signature, f)
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

func generateReadArray(tp Type, isUnsafe bool, f *os.File) {
	unsafePfx := ""
	if isUnsafe {
		unsafePfx = "WithoutType"
	}
	line(fmt.Sprintf("func (buf *IgniteBuffer) Read%sArray%s() []%s {", tp.Name, unsafePfx, tp.Name), f)
	line(TAB+"l := int(buf.ReadInt32())", f)
	line(TAB+fmt.Sprintf("res := make([]%s, l)", tp.Name), f)
	line(TAB+"for i := 0; i < l; i++ {", f)
	line(TAB+TAB+fmt.Sprintf("res[i] = buf.Read%s()", tp.Name), f)
	line(TAB+"}", f)
	line(TAB+"return res", f)
	line("}", f)
}

func generateConstructor(request *Type, typeName string, f *os.File) bool {
	constructorRequired := false
	for _, fld := range request.Fields {
		constructorRequired = constructorRequired || fld.Value != ""
	}

	if !constructorRequired {
		return false
	}

	line(fmt.Sprintf("func %s() %s {", constructorName(typeName), typeName), f)

	var dfltVals string

	forEachField(request, func(fld Field) {
		if fld.Value != "" {
			if dfltVals != "" {
				dfltVals += ", "
			}
			dfltVals += fmt.Sprintf("%s: %s", exportedName(fld.Name), defaultValue(fld))
		}
	})

	line(fmt.Sprintf(TAB+"return %s{%s}", typeName, dfltVals), f)
	line("}", f)

	return true
}

func generateWrite(request *Type, typeName string, f *os.File) {
	line(fmt.Sprintf("func (req %s) Write(buf *IgniteBuffer) {", typeName), f)

	forEachField(request, func(fld Field) {
		line(fmt.Sprintf(TAB+"buf.%s(req.%s)", writeMethodName(fld.Type, &fld), exportedName(fld.Name)), f)
	})

	line("}", f)
}

func generateWriteArray(tp Type, unsafe bool, f *os.File) {
	unsafePfx := ""
	if unsafe {
		unsafePfx = "WithoutType"
	}
	line(fmt.Sprintf("func (buf *IgniteBuffer) Write%sArray%s(req []%s) {", tp.Name, unsafePfx, tp.Name), f)
	line(TAB+"l := len(req)", f)
	line(TAB+"buf.WriteInt32(int32(l))", f)
	line(TAB+"for i := 0; i < l; i++ {", f)
	line(TAB+TAB+"req[i].Write(buf)", f)
	line(TAB+"}", f)
	line("}", f)
}

func generateMapWrite(keyType string, valType string, f *os.File) {
	goKeyType := typeMap[keyType]
	goValType := typeMap[valType]

	isGoKeyTypePointer := goKeyType[0] == '*'
	goKeyType = removePointer(goKeyType)
	isGoValTypePointer := goValType[0] == '*'
	goValType = removePointer(goValType)

	line(fmt.Sprintf("func (buf *IgniteBuffer) %s(m *map[%s]%s) {", mapMethodName(write, keyType, valType), goKeyType, goValType), f)
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

func generateMapRead(keyType string, valType string, f *os.File) {
	goKeyType := typeMap[keyType]
	goValType := typeMap[valType]

	keyPointer := ""
	if goKeyType[0] == '*' {
		keyPointer = "*"
	}

	valPointer := ""
	if goValType[0] == '*' {
		valPointer = "*"
	}

	goKeyType = removePointer(goKeyType)
	goValType = removePointer(goValType)

	line(fmt.Sprintf("func (buf *IgniteBuffer) %s() *map[%s]%s {", mapMethodName(read, keyType, valType), goKeyType, goValType), f)
	line(TAB+fmt.Sprintf("res := make(map[%s]%s)", goKeyType, goValType), f)
	line(TAB+"l := int(buf.ReadInt32())", f)
	line(TAB+"for i := 0; i < l; i++ {", f)
	line(TAB+TAB+fmt.Sprintf("key := %sbuf.%s()", keyPointer, methodName(read, keyType)), f)
	line(TAB+TAB+fmt.Sprintf("res[key] = %sbuf.%s()", valPointer, methodName(read, valType)), f)
	line(TAB+"}", f)
	line(TAB+"return &res", f)

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

func mapMethodName(pfx string, keyType string, valType string) string {
	keyPart := methodPartName(keyType)
	valPart := methodPartName(valType)

	return fmt.Sprintf("%s%s%s", pfx, keyPart, valPart)
}

func writeMethodName(tp string, fld *Field) string {
	if fld != nil && fld.Type == "map" {
		return mapMethodName(write, fld.KeyType, fld.ValueType)
	}

	res := methodName(write, tp)
	if fld != nil && fld.Unsafe {
		return res + withoutTypeSuffix
	}
	return res
}

func readMethodName(fld Field) string {
	if fld.Type == "map" {
		return mapMethodName(read, fld.KeyType, fld.ValueType)
	}

	res := methodName(read, fld.Type)
	if fld.Unsafe {
		return res + withoutTypeSuffix
	}
	return res
}

func methodName(pfx string, tp string) string {
	return pfx + methodPartName(tp)
}

func methodPartName(tp string) string {
	if isArrayType(tp) {
		return methodPartName(arrayComponentType(tp)) + "Array"
	}

	tpPart, contains := simpleName[tp]
	if !contains {
		return exportedName(removePointer(goType(tp, nil)))
	}
	return tpPart
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
		return fmt.Sprintf("*map[%s]%s",
			removePointer(goType(fld.KeyType, nil)),
			removePointer(goType(fld.ValueType, nil)))
	}

	res, ok := typeMap[tp]
	if !ok {
		res, ok = customTypes[tp]
		if !ok {
			panic(fmt.Sprintf("unknown type %s", tp))
		}
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

func forEachField(tp *Type, consumer func(fld Field)) {
	for _, fld := range tp.Fields {
		consumer(fld)
	}

	for _, fld := range tp.Optional {
		consumer(fld)
	}
}

func isArrayType(keyType string) bool {
	return strings.HasSuffix(keyType, arraySuffix)
}

func arrayComponentType(tp string) string {
	return tp[:len(tp)-len(arraySuffix)]
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}
