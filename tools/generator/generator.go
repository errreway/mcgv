package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"golang.org/x/tools/imports"
	"os"
	"path/filepath"
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
// This file contains pieces of code from https://github.com/iancoleman/strcase under MIT license
/*
	The MIT License (MIT)

	Copyright (c) 2015 Ian Coleman
	Copyright (c) 2018 Ma_124, <github.com/Ma124>

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, Subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or Substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
*/

var buildTags = flag.String("build_tags", "", "build tags to add to generated file")
var specifiedName = flag.String("output_filename", "", "specify the filename of the output")
var processPkg = flag.Bool("pkg", false, "process the whole package instead of just the given file")
var writeRegisterFunc = flag.Bool("with_register_func", false, "write register function to file")

const (
	ignPackage = "gitverse.ru/sbertech/ignite-go-client"
)

func generate(fname string) (err error) {
	fInfo, err := os.Stat(fname)
	if err != nil {
		return err
	}

	p := Parser{
		FileName:       fname,
		IsDir:          fInfo.IsDir(),
		Aliases:        make(map[string]string),
		ReverseAliases: make(map[string]string),
	}
	if err := p.Parse(); err != nil {
		return fmt.Errorf("error parsing %v: %v", p.FileName, err)
	}

	var outName string
	if *specifiedName != "" {
		outName = *specifiedName
	} else if p.IsDir {
		outName = filepath.Join(p.FileName, p.PkgName+"_binarylizable.go")
	} else {
		if s := strings.TrimSuffix(p.FileName, ".go"); s == p.FileName {
			return errors.New("filename must end in '.go'")
		} else {
			outName = s + "_binarylizable.go"
		}
	}

	if *buildTags != "" {
		p.BuildTags = strings.TrimSpace(*buildTags)
	}

	g := generator{p}
	err = g.printFile(outName)
	return
}

type generator struct {
	Parser
}

func (g *generator) printStructs() (*bytes.Buffer, error) {
	outbuf := bytes.NewBuffer(make([]byte, 0, 4096))
	g.writePkgHeader(outbuf)

	var myImports []string
	hasIgnitePackage := false
	for _, imp := range g.Imports {
		unqPath, _ := strconv.Unquote(imp.Path.Value)
		if unqPath == ignPackage {
			hasIgnitePackage = true
		}
		if imp.Name != nil {
			// have an alias, include it.
			myImports = append(myImports, imp.Name.Name+` `+imp.Path.Value)
		} else {
			myImports = append(myImports, imp.Path.Value)
		}
	}
	if !hasIgnitePackage {
		myImports = append(myImports, ignPackage)
	}
	writeImportHeader(outbuf, myImports...)

	for _, strct := range g.Structs {
		if strct.TypeName != strct.Name {
			outbuf.WriteString("func (o *" + strct.Name + ") TypeName() string {\n")
			outbuf.WriteString("return \"" + strct.TypeName + "\"\n")
			outbuf.WriteString("}\n\n")
		}

		outbuf.WriteString(fmt.Sprintf("func (o *%s) Platform() %s{\n", strct.Name, g.getAliasedGoType("ignite", "MarshallerPlatform")))
		var marshStr string
		switch strct.Platform {
		case ignite.DotNetMarshaller:
			marshStr = g.getAliasedGoType("ignite", "DotNetMarshaller")
		default:
			marshStr = g.getAliasedGoType("ignite", "JavaMarshaller")
		}
		outbuf.WriteString(fmt.Sprintf("return %s\n", marshStr))
		outbuf.WriteString("}\n\n")

		g.writeWriter(outbuf, strct)
		g.writeReader(outbuf, strct)
	}
	if *writeRegisterFunc {
		g.writeRegisterFunction(outbuf)
	}
	return outbuf, nil
}

func (g *generator) writeWriter(b *bytes.Buffer, strct IgniteStruct) {
	affinityKey := ""
	b.WriteString(fmt.Sprintf("func (o * %s) Write(ctx %s, writer %s) error {\n", strct.Name,
		g.getAliasedGoType("context", "Context"), g.getAliasedGoType("ignite", "BinaryWriter")))
	for _, field := range strct.Fields {
		alias := field.Alias
		if len(alias) == 0 {
			alias = field.Name
		}
		fldType := field.Type
		checkForNil := fldType.IsPointer() && fldType.IsScalar()
		if checkForNil {
			b.WriteString("if o." + field.Name + " == nil {\n")
			writeErrorCheck(b, fmt.Sprintf("writer.WriteNullField(ctx, \"%s\", %s)",
				alias, g.getAliasedGoType("ignite", fldType.IgniteType().String())))
			b.WriteString("} else {\n")
		}
		if checkForNil && fldType.IgniteType() != ignite.DecimalType {
			writeErrorCheck(b, fmt.Sprintf("writer.WriteField(ctx, \"%s\", *o.%s)", alias, field.Name))
		} else {
			writeErrorCheck(b, fmt.Sprintf("writer.WriteField(ctx, \"%s\", o.%s)", alias, field.Name))
		}
		if checkForNil {
			b.WriteString("}\n")
		}
		if field.IsAffinityKey {
			affinityKey = alias
		}
	}
	b.WriteString("return nil\n")
	b.WriteString("}\n\n")

	if len(affinityKey) > 0 {
		b.WriteString("func (o *" + strct.Name + ") AffinityKeyName() string {\n")
		b.WriteString("return \"" + affinityKey + "\"\n")
		b.WriteString("}\n\n")
	}
}

func (g *generator) writeReader(b *bytes.Buffer, strct IgniteStruct) {
	b.WriteString(fmt.Sprintf("func (o *%s) Read(ctx %s, reader %s) error {\n", strct.Name,
		g.getAliasedGoType("context", "Context"), g.getAliasedGoType("ignite", "BinaryReader")))
	for _, field := range strct.Fields {
		g.writeReadField(b, &field)
	}
	b.WriteString("return nil\n")
	b.WriteString("}\n\n")
}

func (g *generator) writeReadField(b *bytes.Buffer, field *IgniteField) {
	alias := field.Alias
	if len(alias) == 0 {
		alias = field.Name
	}
	fldType := field.Type
	if fldType.IsArray() {
		elType := fldType.ElementType()
		if isPointerArray(fldType.IgniteType()) && !elType.IsPointer() {
			b.WriteString("{\n")
			b.WriteString(fmt.Sprintf("var tmp []*%s\n", elType.GoType()))
			writeErrorCheck(b, fmt.Sprintf("reader.ReadField(ctx, \"%s\", &tmp)", alias))
			b.WriteString("if tmp != nil {\n")
			b.WriteString(fmt.Sprintf("o.%s = make([]%s, 0, len(tmp))\n", field.Name, elType.GoType()))
			b.WriteString("for _, pel := range tmp {\n")
			b.WriteString(fmt.Sprintf("var el %s\n", elType.GoType()))
			b.WriteString("if pel != nil {\n el = *pel \n }\n")
			b.WriteString(fmt.Sprintf("o.%s = append(o.%s, el)\n", field.Name, field.Name))
			b.WriteString("}\n}\n}\n")
			return
		}
		if isPrimitiveArray(fldType.IgniteType()) && elType.GoType() != elType.OriginalGoType() {
			b.WriteString("{\n")
			b.WriteString(fmt.Sprintf("var tmp []%s\n", elType.GoType()))
			writeErrorCheck(b, fmt.Sprintf("reader.ReadField(ctx, \"%s\", &tmp)", alias))
			b.WriteString("if tmp != nil {\n")
			b.WriteString(fmt.Sprintf("o.%s = make([]%s, 0, len(tmp))\n", field.Name, elType.OriginalGoType()))
			b.WriteString("for _, el := range tmp {\n")
			b.WriteString(fmt.Sprintf("o.%s = append(o.%s, %s(el))\n", field.Name, field.Name, elType.OriginalGoType()))
			b.WriteString(fmt.Sprintf("\n}\n} else {\n o.%s = nil \n}\n", field.Name))
			b.WriteString("}\n")
			return
		}
		if fldType.IgniteType() == ignite.ObjectArrayType && !elType.IsInterface() {
			b.WriteString("{\n")
			b.WriteString("var tmp []interface{}\n")
			writeErrorCheck(b, fmt.Sprintf("reader.ReadField(ctx, \"%s\", &tmp)", alias))
			b.WriteString("if tmp != nil {\n")
			b.WriteString(fmt.Sprintf("o.%s = make([]%s, 0, len(tmp))\n", field.Name, elType.OriginalGoType()))
			b.WriteString("for _, el0 := range tmp {\n")
			b.WriteString(fmt.Sprintf("var el %s = nil\n", elType.OriginalGoType()))
			b.WriteString("if el0 != nil {\n")
			b.WriteString("var ok bool\n")
			b.WriteString(fmt.Sprintf("el, ok = el0.(%s)\n", elType.GoType()))
			b.WriteString(fmt.Sprintf("if !ok {\n return %s(\"failed to convert %%v to %s\", el0) \n}\n",
				g.getAliasedGoType("fmt", "Errorf"), elType.OriginalGoType()))
			b.WriteString("}\n")
			b.WriteString(fmt.Sprintf("o.%s = append(o.%s, el)\n", field.Name, field.Name))
			b.WriteString("}\n}\n}\n")
			return
		}
	}
	if fldType.IsPrimitive() && fldType.GoType() != fldType.OriginalGoType() {
		b.WriteString("{\n")
		b.WriteString(fmt.Sprintf("var tmp %s\n", fldType.GoType()))
		writeErrorCheck(b, fmt.Sprintf("reader.ReadField(ctx, \"%s\", &tmp)", alias))
		if fldType.IsPointer() {
			b.WriteString("if tmp != nil {\n")
			b.WriteString(fmt.Sprintf("tmp0 := %s(*tmp)\n", fldType.ElementType().OriginalGoType()))
			b.WriteString(fmt.Sprintf("o.%s = &tmp0\n", field.Name))
			b.WriteString(fmt.Sprintf("} else {\n o.%s = nil \n}\n", field.Name))
		} else {
			b.WriteString(fmt.Sprintf("o.%s = %s(tmp)\n", field.Name, fldType.OriginalGoType()))
		}
		b.WriteString("}\n")
		return
	}
	if fldType.IsMap() {
		b.WriteString("{\n")
		b.WriteString("var tmp ignite.Map\n")
		writeErrorCheck(b, fmt.Sprintf("reader.ReadField(ctx, \"%s\", &tmp)", alias))
		b.WriteString("if !tmp.IsNull() {\n")
		b.WriteString("var err error\n")
		b.WriteString(fmt.Sprintf("o.%s, err = %s[%s, %s](tmp)\n",
			field.Name, g.getAliasedGoType("ignite", "ToMap"), fldType.ElementKeyType().OriginalGoType(), fldType.ElementType().OriginalGoType()))
		b.WriteString("if err != nil {\n return err \n}\n")
		b.WriteString(fmt.Sprintf("} else {\n o.%s = nil \n}\n}\n", field.Name))
		return
	}
	writeErrorCheck(b, fmt.Sprintf("reader.ReadField(ctx, \"%s\", &o.%s)", alias, field.Name))
}

func writeErrorCheck(b *bytes.Buffer, expr string) {
	b.WriteString(fmt.Sprintf("if err := %s; err != nil {\n return err \n}\n", expr))
}

func (g *generator) writePkgHeader(b *bytes.Buffer) {
	if len(g.BuildTags) > 0 {
		b.WriteString("// go:build " + g.BuildTags + "\n")
	}
	b.WriteString("package ")
	b.WriteString(g.PkgName)
	b.WriteByte('\n')
	b.WriteString("// Code generated by gitverse.ru/sbertech/ignite-go-client/tools/generator DO NOT EDIT.\n\n")
}

func writeImportHeader(b *bytes.Buffer, imports ...string) {
	b.WriteString("import (\n")
	for _, im := range imports {
		if im[len(im)-1] == '"' {
			// support aliased imports
			_, _ = fmt.Fprintf(b, "\t%s\n", im)
		} else {
			_, _ = fmt.Fprintf(b, "\t%q\n", im)
		}
	}
	b.WriteString(")\n\n")
}

func (g *generator) writeRegisterFunction(b *bytes.Buffer) {
	structs := g.Structs
	if len(structs) == 0 {
		return
	}
	var funcName string
	if len(structs) == 1 {
		funcName = fmt.Sprintf("Register%s", toCamelCase(structs[0].Name))
	} else {
		if !g.IsDir {
			funcName = strings.TrimSuffix(g.FileName, ".go")
		} else {
			funcName = g.PkgName
		}
		funcName = fmt.Sprintf("Register%sIgniteTypes", toCamelCase(funcName))
	}
	b.WriteString(fmt.Sprintf("func %s(cli *%s) {\n", funcName, g.getAliasedGoType("ignite", "Client")))
	for _, ignStruct := range structs {
		b.WriteString(fmt.Sprintf("cli.RegisterBinarylizable(func() %s {\n", g.getAliasedGoType("ignite", "Binarylizable")))
		b.WriteString(fmt.Sprintf("return &%s{}\n", ignStruct.Name))
		b.WriteString("})\n")
	}
	b.WriteString("}\n")
}

func (g *generator) printFile(file string) error {
	out, err := g.printStructs()
	if err != nil {
		return err
	}
	res := goformat(file, out.Bytes())
	err = <-res
	if err != nil {
		_ = os.WriteFile(file+".broken", out.Bytes(), os.ModePerm)
		return err
	}
	return nil
}

func format(file string, data []byte) error {
	out, err := imports.Process(file, data, nil)
	if err != nil {
		return err
	}
	return os.WriteFile(file, out, 0o600)
}

func goformat(file string, data []byte) <-chan error {
	out := make(chan error, 1)
	go func(file string, data []byte, end chan error) {
		end <- format(file, data)
	}(file, data, out)
	return out
}

func toCamelCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := true
	prevIsCap := false
	for _, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if prevIsCap && vIsCap {
			v += 'a'
			v -= 'A'
		}
		prevIsCap = vIsCap

		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.' || v == '/' || v == '\\'
		}
	}
	return n.String()
}

func main() {
	flag.Parse()
	files := flag.Args()

	gofile := os.Getenv("GOFILE")
	if *processPkg {
		gofile = filepath.Dir(gofile)
	}

	if len(files) == 0 && gofile != "" {
		files = []string{gofile}
	} else if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, fname := range files {
		if err := generate(fname); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
