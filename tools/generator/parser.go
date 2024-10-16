package main

import (
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// This file contains pieces of code from https://github.com/tinylib/msgp under MIT license
/*
	Copyright (c) 2014 Philip Hofer
	Portions Copyright (c) 2009 The Go Authors (license at http://golang.org) where indicated

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"),
	to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute,
	sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so,
	subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
	INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
	DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

const (
	structComment = "ignite:binarylizable"
	enumComment   = "ignite:enum"
)

type Parser struct {
	PkgName        string
	BuildTags      string
	Structs        []IgniteStruct
	Enums          map[string]*IgniteEnum
	Imports        []*ast.ImportSpec
	Aliases        map[string]string
	ReverseAliases map[string]string
	FileName       string
	IsDir          bool
	Error          error
}

type IgniteEnum struct {
	Name     string
	TypeName string
	Platform ignite.MarshallerPlatform
	Values   map[string]EnumValue
}

type EnumValue struct {
	Name   string
	GoName string
}

type IgniteStruct struct {
	Fields   []IgniteField
	Name     string
	TypeName string
	Platform ignite.MarshallerPlatform
}

type IgniteField struct {
	Name          string
	Alias         string
	IsAffinityKey bool
	Type          IgniteType
}

type visitor struct {
	*Parser
	name     string
	typeName string
	platform ignite.MarshallerPlatform
}

var typeNameRe = regexp.MustCompile(`typename="([^"]+)"`)
var marshPlatformRe = regexp.MustCompile(`platform="([^"]+)"`)
var enumNameRe = regexp.MustCompile(`name="([^"]+)"`)
var wellKnownPackages = map[string]string{
	"gitverse.ru/sbertech/ignite-go-client": "ignite",
	"github.com/cockroachdb/apd/v3":         "apd",
	"github.com/google/uuid":                "uuid",
	"time":                                  "time",
	"context":                               "context",
	"fmt":                                   "fmt",
}

type typeInfo struct {
	typeName string
	platform ignite.MarshallerPlatform
	isEnum   bool
}

func (p *Parser) shouldSkip(comments *ast.CommentGroup) (shouldSkip bool, tInfo typeInfo) {
	tInfo = typeInfo{
		typeName: "",
		platform: ignite.JavaMarshaller,
	}
	shouldSkip = true
	if comments == nil {
		return
	}
	for _, v := range comments.List {
		comment := v.Text
		if len(comment) > 2 {
			switch comment[1] {
			case '/':
				// -style comment (no newline at the end)
				comment = comment[2:]
			case '*':
				/*-style comment */
				comment = comment[2 : len(comment)-2]
			}
		}
		for _, comment := range strings.Split(comment, "\n") {
			comment = strings.TrimSpace(comment)
			if strings.HasPrefix(comment, structComment) || strings.HasPrefix(comment, enumComment) {
				tInfo.isEnum = strings.HasPrefix(comment, enumComment)
				shouldSkip = false
				match := typeNameRe.FindStringSubmatch(comment)
				if len(match) > 0 {
					tInfo.typeName = match[1]
				}
				match = marshPlatformRe.FindStringSubmatch(comment)
				if len(match) > 0 {
					switch strings.ToLower(match[1]) {
					case "dotnet":
						tInfo.platform = ignite.DotNetMarshaller
					case "java":
						tInfo.platform = ignite.JavaMarshaller
					}
				}
			}
		}
	}
	return
}

func (v *visitor) Visit(n ast.Node) (w ast.Visitor) {
	if v.Error != nil {
		return nil
	}
	switch n := n.(type) {
	case *ast.Package:
		return v
	case *ast.File:
		v.PkgName = n.Name.String()
		for _, imp := range n.Imports {
			if imp.Name != nil {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					path = imp.Path.Value
				}
				v.Aliases[imp.Name.Name] = path
				if _, ok := wellKnownPackages[path]; ok {
					v.ReverseAliases[wellKnownPackages[path]] = imp.Name.Name
				}
			}
		}
		v.Imports = append(v.Imports, n.Imports...)
		return v
	case *ast.GenDecl:
		shouldSkip, _ := v.shouldSkip(n.Doc)
		if !shouldSkip {
			for _, nc := range n.Specs {
				switch nct := nc.(type) {
				case *ast.TypeSpec:
					nct.Doc = n.Doc
				}
			}
		}
		return v
	case *ast.TypeSpec:
		shouldSkip, tInfo := v.shouldSkip(n.Doc)
		if shouldSkip {
			break
		}
		v.name = n.Name.String()
		typeName := tInfo.typeName
		if len(typeName) > 0 {
			v.typeName = typeName
		} else {
			v.typeName = v.name
		}
		v.platform = tInfo.platform
		if tInfo.isEnum {
			typeIdent, ok := n.Type.(*ast.Ident)
			if ok && strings.Contains(typeIdent.Name, "int") {
				v.Enums[v.name] = &IgniteEnum{
					Name:     v.name,
					TypeName: v.typeName,
					Platform: v.platform,
					Values:   make(map[string]EnumValue),
				}
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "skipped enum %s: invalid type %s\n", v.name, typeIdent.Name)
			}
		}
		return v
	case *ast.StructType:
		strct := IgniteStruct{
			Name:     v.name,
			TypeName: v.typeName,
			Fields:   make([]IgniteField, 0),
			Platform: v.platform,
		}
		strct.Fields = v.parseFieldList(n.Fields)
		v.Structs = append(v.Structs, strct)
	case *ast.ValueSpec:
		if typeIdent, ok := n.Type.(*ast.Ident); ok {
			v.name = typeIdent.Name
		}
		enum := v.Enums[v.name]
		if enum == nil {
			break
		}
		for _, nameIdent := range n.Names {
			name := nameIdent.Name
			if name == "_" {
				continue
			}
			enumName := name
			if n.Comment != nil {
				commentName := v.getEnumName(n.Comment.Text())
				if len(commentName) > 0 {
					enumName = commentName
				}
			}
			enum.Values[name] = EnumValue{
				GoName: name,
				Name:   enumName,
			}
		}
	}
	return nil
}

func (v *visitor) parseFieldList(fl *ast.FieldList) []IgniteField {
	if fl == nil || fl.NumFields() == 0 {
		return nil
	}
	out := make([]IgniteField, 0, fl.NumFields())
	for _, field := range fl.List {
		fds := v.getField(field)
		if len(fds) > 0 {
			out = append(out, fds...)
		}
	}
	return out
}

func (p *Parser) getEnumName(constComment string) string {
	constComment = strings.TrimSpace(constComment)
	match := enumNameRe.FindStringSubmatch(constComment)
	if len(match) > 0 {
		return match[1]
	}
	return ""
}

func (v *visitor) getField(f *ast.Field) []IgniteField {
	sf := make([]IgniteField, 1)
	// parse tag; otherwise field name is field tag
	if f.Tag != nil {
		body := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("ign")
		if body == "" {
			body = reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("ignite")
		}
		tags := strings.Split(body, ",")
		if len(tags) >= 2 && tags[1] == "affinityKey" {
			sf[0].IsAffinityKey = true
		}
		// ignore "-" fields
		if tags[0] == "-" {
			return nil
		}
		if tags[0] == "affinityKey" {
			sf[0].IsAffinityKey = true
		} else {
			sf[0].Alias = tags[0]
		}
	}

	ignTyp, ok := v.parseType(f.Type)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "skipped %s field of struct %s\n", f.Names[0].Name, v.name)
		return nil
	}
	sf[0].Type = ignTyp

	// parse field name
	switch len(f.Names) {
	case 0:
		// Doesn't support embedded struct
		return nil
	case 1:
		if len(sf[0].Name) == 0 {
			sf[0].Name = f.Names[0].Name
		}
	default:
		// this is for a multiple in-line declaration,
		// e.g. type A struct { One, Two int }
		sf[0].IsAffinityKey = false
		sf = sf[0:0]
		for _, nm := range f.Names {
			sf = append(sf, IgniteField{
				Name: nm.Name,
				Type: ignTyp,
			})
		}
		return sf
	}

	return sf
}

func (p *Parser) Parse() error {
	fset := token.NewFileSet()
	if p.IsDir {
		packages, err := parser.ParseDir(fset, p.FileName, excludeTestFiles, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, pckg := range packages {
			ast.Walk(&visitor{Parser: p}, pckg)
		}
	} else {
		f, err := parser.ParseFile(fset, p.FileName, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		ast.Walk(&visitor{Parser: p}, f)
	}
	return p.Error
}

func excludeTestFiles(fi os.FileInfo) bool {
	return !strings.HasSuffix(fi.Name(), "_test.go")
}

type IgniteType interface {
	IgniteType() ignite.TypeDesc
	IsPointer() bool
	IsInterface() bool
	IsPrimitive() bool
	IsIdent() bool
	IsScalar() bool
	IsArray() bool
	IsMap() bool
	ElementType() IgniteType
	ElementKeyType() IgniteType
	GoType() string
	OriginalGoType() string
}

type IdentType struct {
	igniteType     ignite.TypeDesc
	goType         string
	goOriginalType string
}

func (ident *IdentType) IgniteType() ignite.TypeDesc {
	return ident.igniteType
}

func (ident *IdentType) IsPointer() bool {
	return false
}

func (ident *IdentType) IsInterface() bool {
	return false
}

func (ident *IdentType) IsScalar() bool {
	return isScalarType(ident.igniteType)
}

func (ident *IdentType) IsIdent() bool {
	return true
}

func (ident *IdentType) IsPrimitive() bool {
	return isPrimitiveType(ident.igniteType)
}

func (ident *IdentType) IsMap() bool {
	return false
}

func (ident *IdentType) ElementType() IgniteType {
	return ident
}

func (ident *IdentType) ElementKeyType() IgniteType {
	return nil
}

func (ident *IdentType) GoType() string {
	return ident.goType
}

func (ident *IdentType) OriginalGoType() string {
	return ident.goOriginalType
}

func (ident *IdentType) IsArray() bool {
	return false
}

type PtrType struct {
	elem IgniteType
}

func (ptr *PtrType) IgniteType() ignite.TypeDesc {
	return ptr.elem.IgniteType()
}

func (ptr *PtrType) IsPrimitive() bool {
	return ptr.elem.IsPrimitive()
}

func (ptr *PtrType) IsScalar() bool {
	return ptr.elem.IsScalar()
}

func (ptr *PtrType) IsPointer() bool {
	return true
}

func (ptr *PtrType) IsIdent() bool {
	return false
}

func (ptr *PtrType) IsInterface() bool {
	return false
}

func (ptr *PtrType) IsArray() bool {
	return false
}

func (ptr *PtrType) IsMap() bool {
	return false
}

func (ptr *PtrType) ElementKeyType() IgniteType {
	return nil
}

func (ptr *PtrType) ElementType() IgniteType {
	return ptr.elem
}

func (ptr *PtrType) GoType() string {
	return "*" + ptr.elem.GoType()
}

func (ptr *PtrType) OriginalGoType() string {
	return "*" + ptr.elem.OriginalGoType()
}

type ArrayType struct {
	igniteType ignite.TypeDesc
	elType     IgniteType
}

func (arr *ArrayType) IgniteType() ignite.TypeDesc {
	return arr.igniteType
}

func (arr *ArrayType) IsPrimitive() bool {
	return false
}

func (arr *ArrayType) IsIdent() bool {
	return false
}

func (arr *ArrayType) IsScalar() bool {
	return false
}

func (arr *ArrayType) IsPointer() bool {
	return false
}

func (arr *ArrayType) IsInterface() bool {
	return false
}

func (arr *ArrayType) IsArray() bool {
	return true
}

func (arr *ArrayType) IsMap() bool {
	return false
}

func (arr *ArrayType) ElementKeyType() IgniteType {
	return nil
}

func (arr *ArrayType) ElementType() IgniteType {
	return arr.elType
}

func (arr *ArrayType) GoType() string {
	return "[]" + arr.elType.GoType()
}

func (arr *ArrayType) OriginalGoType() string {
	return "[]" + arr.elType.OriginalGoType()
}

type InterfaceType struct{}

func (i *InterfaceType) IgniteType() ignite.TypeDesc {
	return -1
}

func (i *InterfaceType) IsIdent() bool {
	return false
}

func (i *InterfaceType) IsScalar() bool {
	return false
}

func (i *InterfaceType) IsPrimitive() bool {
	return false
}

func (i *InterfaceType) IsPointer() bool {
	return false
}

func (i *InterfaceType) IsInterface() bool {
	return true
}

func (i *InterfaceType) IsArray() bool {
	return false
}

func (i *InterfaceType) IsMap() bool {
	return false
}

func (i *InterfaceType) ElementKeyType() IgniteType {
	return nil
}

func (i *InterfaceType) ElementType() IgniteType {
	return nil
}

func (i *InterfaceType) GoType() string {
	return "interface{}"
}

func (i *InterfaceType) OriginalGoType() string {
	return i.GoType()
}

type MapType struct {
	keyType   IgniteType
	valueType IgniteType
}

func (m *MapType) IgniteType() ignite.TypeDesc {
	return ignite.MapType
}

func (m *MapType) IsPointer() bool {
	return false
}

func (m *MapType) IsInterface() bool {
	return false
}

func (m *MapType) IsPrimitive() bool {
	return false
}

func (m *MapType) IsIdent() bool {
	return false
}

func (m *MapType) IsScalar() bool {
	return false
}

func (m *MapType) IsArray() bool {
	return false
}

func (m *MapType) IsMap() bool {
	return true
}

func (m *MapType) ElementType() IgniteType {
	return m.valueType
}

func (m *MapType) ElementKeyType() IgniteType {
	return m.keyType
}

func (m *MapType) GoType() string {
	return "map[" + m.keyType.OriginalGoType() + "]" + m.valueType.OriginalGoType()
}

func (m *MapType) OriginalGoType() string {
	return m.GoType()
}

func (p *Parser) parseIgniteIdentType(t string) (fType IgniteType) {
	switch t {
	case "any":
		fType = &InterfaceType{}
	case "bool":
		fType = &IdentType{ignite.BoolType, t, t}
	case "uint8", "byte":
		fType = &IdentType{ignite.ByteType, "int8", t}
	case "int8":
		fType = &IdentType{ignite.ByteType, t, t}
	case "uint16":
		fType = &IdentType{ignite.CharType, t, t}
	case "int16":
		fType = &IdentType{ignite.ShortType, t, t}
	case "uint32", "int32":
		fType = &IdentType{ignite.IntType, "int32", t}
	case "uint64", "int64", "uint", "int":
		fType = &IdentType{ignite.LongType, "int64", t}
	case "float32":
		fType = &IdentType{ignite.FloatType, t, t}
	case "float64":
		fType = &IdentType{ignite.DoubleType, t, t}
	case "string":
		fType = &IdentType{ignite.StringType, t, t}
	case "uuid.UUID":
		t := p.getAliasedGoType("uuid", "UUID")
		fType = &IdentType{ignite.UuidType, t, t}
	case "apd.Decimal":
		t := p.getAliasedGoType("apd", "Decimal")
		fType = &IdentType{ignite.DecimalType, t, t}
	case "ignite.Time":
		t := p.getAliasedGoType("ignite", "Time")
		fType = &IdentType{ignite.TimeType, t, t}
	case "ignite.Date":
		t := p.getAliasedGoType("ignite", "Date")
		fType = &IdentType{ignite.DateType, t, t}
	case "time.Time":
		t := p.getAliasedGoType("time", "Time")
		fType = &IdentType{ignite.TimestampType, t, t}
	case "ignite.BinaryObject":
		t := p.getAliasedGoType("ignite", "BinaryObject")
		fType = &IdentType{ignite.BinaryObjectType, t, t}
	case "ignite.Binarylizable":
		t := p.getAliasedGoType("ignite", "Binarylizable")
		fType = &IdentType{ignite.BinaryObjectType, t, t}
	case "ignite.Collection":
		t := p.getAliasedGoType("ignite", "Collection")
		fType = &IdentType{ignite.CollectionType, t, t}
	case "ignite.Map":
		t := p.getAliasedGoType("ignite", "Map")
		fType = &IdentType{ignite.MapType, t, t}
	default:
		fType = &IdentType{-2, t, t}
	}
	return
}

func (p *Parser) getAliasedGoType(pkg string, t string) string {
	if alias, ok := p.ReverseAliases[pkg]; ok {
		return alias + "." + t
	}
	return pkg + "." + t
}

func getArrayType(t IgniteType) (arrType ignite.TypeDesc, ok bool) {
	ok = true
	if t.IsInterface() || t.IgniteType() == ignite.BinaryObjectType {
		arrType = ignite.ObjectArrayType
		return
	}
	switch t.IgniteType() {
	case ignite.BoolType:
		arrType = ignite.BoolArrayType
	case ignite.ByteType:
		arrType = ignite.ByteArrayType
	case ignite.ShortType:
		arrType = ignite.ShortArrayType
	case ignite.CharType:
		arrType = ignite.CharArrayType
	case ignite.IntType:
		arrType = ignite.IntArrayType
	case ignite.LongType:
		arrType = ignite.LongArrayType
	case ignite.FloatType:
		arrType = ignite.FloatArrayType
	case ignite.DoubleType:
		arrType = ignite.DoubleArrayType
	case ignite.StringType:
		arrType = ignite.StringArrayType
	case ignite.UuidType:
		arrType = ignite.UuidArrayType
	case ignite.TimeType:
		arrType = ignite.TimeArrayType
	case ignite.DateType:
		arrType = ignite.DateArrayType
	case ignite.TimestampType:
		arrType = ignite.TimestampArrayType
	case ignite.DecimalType:
		arrType = ignite.DecimalArrayType
	default:
		ok = false
	}
	return
}

func isScalarType(t ignite.TypeDesc) bool {
	switch t {
	case ignite.ByteType, ignite.BoolType, ignite.ShortType, ignite.CharType, ignite.IntType, ignite.LongType,
		ignite.FloatType, ignite.DoubleType, ignite.StringType, ignite.UuidType, ignite.TimeType, ignite.DateType,
		ignite.TimestampType, ignite.DecimalType:
		return true
	default:
		return false
	}
}

func isPrimitiveType(t ignite.TypeDesc) bool {
	switch t {
	case ignite.ByteType, ignite.BoolType, ignite.ShortType, ignite.CharType, ignite.IntType, ignite.LongType,
		ignite.FloatType, ignite.DoubleType:
		return true
	default:
		return false
	}
}

func isPointerArray(t ignite.TypeDesc) bool {
	switch t {
	case ignite.StringArrayType, ignite.UuidArrayType, ignite.DateArrayType, ignite.TimeArrayType,
		ignite.TimestampArrayType, ignite.DecimalArrayType:
		return true
	default:
		return false
	}
}

func isPrimitiveArray(t ignite.TypeDesc) bool {
	switch t {
	case ignite.BoolArrayType, ignite.ByteArrayType, ignite.ShortArrayType, ignite.CharArrayType, ignite.IntArrayType, ignite.LongArrayType, ignite.FloatArrayType, ignite.DoubleArrayType:
		return true
	default:
		return false
	}
}

func (p *Parser) parseType(t ast.Expr) (IgniteType, bool) {
	switch t := t.(type) {
	case *ast.Ident:
		return p.parseIgniteIdentType(t.Name), true
	case *ast.MapType:
		kElt, ok := p.parseType(t.Key)
		if ok && kElt.IsScalar() && !kElt.IsPointer() {
			vElt, ok := p.parseType(t.Value)
			if ok {
				return &MapType{
					keyType:   kElt,
					valueType: vElt,
				}, true
			}
		}
	case *ast.ArrayType:
		elt, ok := p.parseType(t.Elt)
		if ok {
			arrType, ok := getArrayType(elt)
			if ok {
				if isPrimitiveArray(arrType) && elt.IsPointer() {
					return nil, false
				}
				if arrType == ignite.ByteArrayType {
					goType := "byte"
					goOriginalType := goType
					if elt.OriginalGoType() == "int8" {
						goOriginalType = elt.OriginalGoType()
					}
					elt = &IdentType{
						igniteType:     ignite.ByteType,
						goType:         goType,
						goOriginalType: goOriginalType,
					}
				}
				return &ArrayType{
					igniteType: arrType,
					elType:     elt,
				}, true
			} else if elt.IsPointer() {
				return &ArrayType{
					igniteType: ignite.ObjectArrayType,
					elType:     elt,
				}, true
			} else {
				return &ArrayType{
					igniteType: ignite.EnumArrayType,
					elType:     elt,
				}, true
			}
		}
	case *ast.StarExpr:
		elType, _ := p.parseType(t.X)
		if elType != nil && !elType.IsArray() && !elType.IsInterface() && !elType.IsPointer() {
			return &PtrType{elem: elType}, true
		}
	case *ast.SelectorExpr:
		firstPart, ok := t.X.(*ast.Ident)
		if ok {
			pkgName := firstPart.Name
			if _, ok = p.Aliases[pkgName]; ok {
				pkgName = p.Aliases[pkgName]
				if _, ok := wellKnownPackages[pkgName]; ok {
					pkgName = wellKnownPackages[pkgName]
				}
			}
			return p.parseIgniteIdentType(pkgName + "." + t.Sel.Name), true
		}
	case *ast.InterfaceType:
		return &InterfaceType{}, true
	}
	return nil, false
}
