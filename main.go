package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var (
	structName     = flag.String("s", "", "结构名词")
	isOverWrite    = flag.Bool("m", false, "是否覆盖")
	output         = flag.String("o", "/tmp/xx.go", "输出位置")
	apiMap         = make(map[string]bool)
	structMap      = make(map[string]bool)
	hasStuctInFile = false
)

//go:generate go version
func main() {
	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	if *structName == "" {
		fmt.Printf("请输入结构名!")
	}
	forStruct := *structName
	wd, _ := os.Getwd()
	file := os.Getenv("GOFILE")
	pack := os.Getenv("GOPACKAGE")

	path := wd + string(os.PathSeparator) + file

	fmt.Printf("wd %s file %s pack %s path %s \r\n", wd, file, pack, path)
	fset := token.NewFileSet()
	f, err := decorator.ParseFile(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}

	dstMethods := make([]*dst.Field, 0)

	dst.Inspect(f, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.GenDecl:
			if x.Tok == token.TYPE {
				if val, ok := x.Specs[0].(*dst.TypeSpec); ok {
					if _, ok := val.Type.(*dst.InterfaceType); ok {
						apiMap[val.Name.Name] = true
					}
					structMap[val.Name.Name] = true
				}
			}
		case *dst.FuncDecl:
			faceDecl := dst.Clone(x).(*dst.FuncDecl)
			if x.Recv != nil {

				if val, ok := faceDecl.Recv.List[0].Type.(*dst.StarExpr); ok {
					if forStruct != val.X.(*dst.Ident).Name {
						return true
					}
				} else if val, ok := x.Recv.List[0].Type.(*dst.Ident); ok {
					if forStruct != val.Name {
						return true
					}
				} else {
					return true
				}
				if faceDecl.Name.Name[:1] == strings.ToLower(faceDecl.Name.Name[:1]) {
					// 私有方法不生成接口
					return true
				}

				dstMethods = append(dstMethods, &dst.Field{
					Names: append(make([]*dst.Ident, 0), &dst.Ident{Name: faceDecl.Name.Name}),
					Type: &dst.FuncType{
						Params:  faceDecl.Type.Params,
						Results: faceDecl.Type.Results,
					},
				})

			}
		}
		return true
	})

	if _, ok := apiMap["I"+forStruct]; ok {
		fmt.Printf("已经存在接口 %s 执行终止操作!", "I"+forStruct)
		return
	}

	if _, ok := structMap[forStruct]; !ok {
		fmt.Printf("不经在Struct %s 执行终止操作!", forStruct)
		return
	}

	apiFace := &dst.GenDecl{
		Tok: token.TYPE,
		Specs: append(make([]dst.Spec, 0), &dst.TypeSpec{
			Name: &dst.Ident{Name: "I" + forStruct},
			Type: &dst.InterfaceType{
				Methods: &dst.FieldList{
					List: dstMethods,
				},
			},
		}),
	}
	isAppend := false
	newDecl := make([]dst.Decl, 0)
	for _, v := range f.Decls {
		if _, ok := v.(*dst.GenDecl); ok {
			if v.(*dst.GenDecl).Tok == token.TYPE {
				// newDecl = append(newDecl, v)
				if !isAppend {
					newDecl = append(newDecl, v, apiFace)
					isAppend = true
					continue
				}

			}
		}
		newDecl = append(newDecl, v)

	}
	f.Decls = newDecl
	f.Decls = newDecl
	tempOut := *output
	if *isOverWrite {
		tempOut = path
	}
	ret, _ := os.OpenFile(tempOut, os.O_WRONLY|os.O_CREATE, 0666)
	if err := decorator.Fprint(ret, f); err != nil {
		panic(err)
	}
	// dst.Print(dsTree)

}
