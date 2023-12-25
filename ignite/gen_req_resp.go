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
	"byte[]":   "[]byte",
	"short":    "int16",
	"int":      "int32",
	"long":     "int64",
	"string":   "*string",
	"string[]": "[]*string",
	"uuid":     "*uuid.UUID",
}

var customTypes = map[string]string{}

type Field struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	KeyType   string `json:"key_type"`
	ValueType string `json:"value_type""`
	Value     string `json:"value"`
	PropIdx   int    `json:"property_index"`
	Order     int    `json:"order"`
}

type Type struct {
	Name       string  `json:"name"`
	PropFields bool    `json:"has_property_fields"`
	Fields     []Field `json:"fields"`
	Optional   []Field `json:"optional"`
}

type Format struct {
	Name         string `json:"name"`
	Code         int16  `json:"code"`
	Request      *Type  `json:"request"`
	Response     *Type  `json:"response"`
	FailResponse *Type  `json:"fail_response"`
}

type TypesList struct {
	Types []Type `json:"types"`
}

func main() {
	serdesDir := "./serdes"

	err := os.MkdirAll(serdesDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	generateTypes(serdesDir)
	generateFormats(serdesDir)
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

		generateWrite(&tp, fmt.Sprintf("func %s(bw BinaryWriter, req *%s) {", writeMethodName(tp.Name, nil), tp.Name), f)
		line("", f)

		generateRead(&tp, tp.Name, fmt.Sprintf("func %s(br BinaryReader) %s {", read+tp.Name, tp.Name), f)
		line("", f)
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
	generateWrite(format.Request, fmt.Sprintf("func (req %s) Write(bw BinaryWriter) {", requestType), f)
	line("", f)
	generateRead(format.Response, responseType, fmt.Sprintf("func (req %s) ReadResponse(br BinaryReader) interface{} {", requestType), f)

	if format.FailResponse != nil {
		failResponseType := exportedName(camelCase(format.Name)) + "FailResponse"
		line("", f)
		generateStruct(format.FailResponse, failResponseType, f)
		line("", f)
		generateRead(format.FailResponse, failResponseType, fmt.Sprintf("func (req %s) ReadFailResponse(br BinaryReader) interface{} {", requestType), f)
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
		if isMapType(fld.Type) {
			generateReadMapField(fld, f)
		} else if isArrayType(fld.Type) {
			generateReadArrayField(fld, f)
		} else {
			generateReadField(fld, f)
		}
	}

	if len(response.Optional) > 0 {
		line("", f)
		line(TAB+"if buf.Available() > 0 {", f)

		for _, fld := range response.Optional {
			if isMapType(fld.Type) {
				generateReadMapField(fld, f)
			} else if isArrayType(fld.Type) {
				generateReadArrayField(fld, f)
			} else {
				generateReadField(fld, f)
			}
		}

		line(TAB+"}", f)
	}

	line(TAB+"return resp", f)
	line("}", f)
}

func generateReadField(fld Field, f *os.File) {
	if _, ok := customTypes[fld.Type]; ok {
		line(fmt.Sprintf(TAB+"resp.%s = %s(br)", exportedName(fld.Name), readMethodName(fld)), f)
	} else {
		line(fmt.Sprintf(TAB+"resp.%s = br.%s()", exportedName(fld.Name), readMethodName(fld)), f)
	}
}

func generateReadArrayField(fld Field, f *os.File) {
	elType := arrayComponentType(fld.Type)
	_, isCustom := customTypes[elType]
	fldName := exportedName(fld.Name)
	line(TAB+fmt.Sprintf("resp.%s = make(%s, int(br.ReadInt32()))", fldName, goType(fld.Type, nil)), f)
	line(TAB+fmt.Sprintf("for i := 0; i < len(resp.%s); i++ {", fldName), f)
	if isCustom {
		line(TAB+TAB+fmt.Sprintf("resp.%s[i] = Read%s(br)", fldName, exportedName(elType)), f)
	} else {
		line(TAB+TAB+fmt.Sprintf("resp.%s[i] = br.Read%s()", fldName, exportedName(elType)), f)
	}
	line(TAB+"}", f)
}

func generateReadMapField(fld Field, f *os.File) {
	fldName := exportedName(fld.Name)
	goKeyType := typeMap[fld.KeyType]
	goValType := typeMap[fld.ValueType]

	isGoKeyTypePointer := goKeyType[0] == '*'

	if _, isKeyCustom := customTypes[fld.KeyType]; isKeyCustom {
		panic(fmt.Sprintf("key cannot be of custom type %s, map field %s", fld.KeyType, fld.Name))
	}

	if _, valCustom := customTypes[fld.ValueType]; valCustom {
		panic(fmt.Sprintf("value cannot be of custom type %s, map field %s", fld.ValueType, fld.Name))
	}

	goKeyType = removePointer(goKeyType)
	line(TAB+fmt.Sprintf("resp.%s = make(map[%s]%s)", fldName, goKeyType, goValType), f)
	goValType = removePointer(goValType)

	line(TAB+"for i := 0; i < int(br.ReadInt32()); i++ {", f)
	line(TAB+TAB+fmt.Sprintf("key := br.Read%s()", exportedName(goKeyType)), f)
	line(TAB+TAB+fmt.Sprintf("val := br.Read%s()", exportedName(goValType)), f)
	if isGoKeyTypePointer {
		line(TAB+TAB+fmt.Sprintf("resp.%s[*key] = val", fldName), f)
	} else {
		line(TAB+TAB+fmt.Sprintf("res.%s[key] = val", fldName), f)
	}
	line(TAB+"}", f)
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

func generateWrite(request *Type, signature string, f *os.File) {
	line(signature, f)

	if request.PropFields {
		line(TAB+"origPos := bw.Position()", f)
		line(TAB+"bw.WriteInt32(0)", f)
		line(TAB+"bw.WriteInt16(0)", f)
		line(TAB+"var propCount int16 = 0", f)
	}

	forEachField(request, func(fld Field) {
		if request.PropFields && fld.PropIdx < 0 {
			return
		}
		if request.PropFields && fld.PropIdx >= 0 {
			line(TAB+fmt.Sprintf("bw.WriteInt16(%d)", fld.PropIdx), f)
		}
		if isMapType(fld.Type) {
			generateWriteMapField(fld, f)
		} else if isArrayType(fld.Type) {
			generateWriteArrayField(fld, f)
		} else {
			generateWriteField(fld, f)
		}

		if request.PropFields && fld.PropIdx >= 0 {
			line(TAB+"propCount += 1", f)
		}
	})

	if request.PropFields {
		line(TAB+"curPos := bw.Position()", f)
		line(TAB+"bw.SetPosition(origPos)", f)
		line(TAB+"bw.WriteInt32(curPos - origPos - IntBytes)", f)
		line(TAB+"bw.WriteInt16(propCount)", f)
		line(TAB+"bw.SetPosition(curPos)", f)
	}

	line("}", f)
}

func generateWriteField(fld Field, f *os.File) {
	_, isCustom := customTypes[fld.Type]

	var fmtString string
	if isCustom {
		fmtString = "%s(bw, &req.%s)"
	} else {
		fmtString = "bw.%s(req.%s)"
	}
	line(fmt.Sprintf(TAB+fmtString, writeMethodName(fld.Type, &fld), exportedName(fld.Name)), f)
}

func generateWriteArrayField(fld Field, f *os.File) {
	fldName := exportedName(fld.Name)
	elType := arrayComponentType(fld.Type)
	_, isCustom := customTypes[elType]

	line(TAB+fmt.Sprintf("bw.WriteInt32(int32(len(req.%s)))", fldName), f)
	line(TAB+fmt.Sprintf("for i := 0; i < len(req.%s); i++ {", fldName), f)

	var fmtString string
	if isCustom {
		fmtString = "Write%s(bw, &req.%s[i])"
	} else {
		fmtString = "bw.Write%s(req.%s[i])"
	}

	line(TAB+TAB+fmt.Sprintf(fmtString, exportedName(elType), fldName), f)
	line(TAB+"}", f)
}

func generateWriteMapField(fld Field, f *os.File) {
	fldName := exportedName(fld.Name)
	goKeyType := typeMap[fld.KeyType]
	goValType := typeMap[fld.ValueType]

	isGoKeyTypePointer := goKeyType[0] == '*'
	goKeyType = removePointer(goKeyType)
	goValType = removePointer(goValType)

	if _, isKeyCustom := customTypes[fld.KeyType]; isKeyCustom {
		panic(fmt.Sprintf("key cannot be of custom type %s, map field %s", fld.KeyType, fld.Name))
	}

	if _, valCustom := customTypes[fld.ValueType]; valCustom {
		panic(fmt.Sprintf("value cannot be of custom type %s, map field %s", fld.ValueType, fld.Name))
	}

	line(TAB+fmt.Sprintf("for key, val := range req.%s {", fldName), f)

	if isGoKeyTypePointer {
		line(TAB+TAB+fmt.Sprintf("bw.Write%s(&key)", exportedName(goKeyType)), f)
	} else {
		line(TAB+TAB+fmt.Sprintf("bw.Write%s(key)", exportedName(goKeyType)), f)
	}
	line(TAB+TAB+fmt.Sprintf("bw.Write%s(val)", exportedName(goValType)), f)

	line(TAB+"}", f)
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
	var res = ""
	if fld != nil && fld.Type == "map" {
		res = mapMethodName(write, fld.KeyType, fld.ValueType)
	} else {
		res = methodName(write, tp)
	}
	return res
}

func readMethodName(fld Field) string {
	var res = ""
	if fld.Type == "map" {
		res = mapMethodName(read, fld.KeyType, fld.ValueType)
	} else {
		res = methodName(read, fld.Type)
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
		return fmt.Sprintf("map[%s]%s",
			removePointer(goType(fld.KeyType, nil)),
			goType(fld.ValueType, nil))
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
	var fields []Field
	var optionals []Field
	if tp.PropFields {
		fields = make([]Field, len(tp.Fields))
		optionals = make([]Field, len(tp.Optional))
		copy(fields, tp.Fields)
		copy(optionals, tp.Optional)

		sort.Slice(fields, func(i int, j int) bool {
			return fields[i].Order < fields[j].Order
		})
	} else {
		fields = tp.Fields
		optionals = tp.Optional
	}

	for _, fld := range fields {
		consumer(fld)
	}

	for _, fld := range optionals {
		consumer(fld)
	}
}

func isArrayType(keyType string) bool {
	return strings.HasSuffix(keyType, arraySuffix)
}

func isMapType(keyType string) bool {
	return keyType == "map"
}

func arrayComponentType(tp string) string {
	return tp[:len(tp)-len(arraySuffix)]
}
