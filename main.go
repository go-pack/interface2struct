package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var (
	stractName = flag.String("s", "", "结构名词")
	apiMap     = make(map[string]bool)
)

//go:generate go version
func main() {
	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	if *stractName == "" {
		fmt.Printf("请输入结构名!")
	}
	forStruct := *stractName
	// *isProxy = true
	// *isOverWrite = true
	wd, _ := os.Getwd()
	file := os.Getenv("GOFILE")
	pack := os.Getenv("GOPACKAGE")

	// wd := "/Users/chen/IdeaProjects/smm-go/internal/services"
	// file := "smmCodeService.go"
	// pack := "daos"
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
				}
			}
		case *dst.FuncDecl:
			faceDecl := dst.Clone(x).(*dst.FuncDecl)
			if x.Recv != nil {
				if forStruct != x.Recv.List[0].Type.(*dst.Ident).Name {
					return true
				}
				dstMethods = append(dstMethods, &dst.Field{
					Names: append(make([]*dst.Ident, 0), &dst.Ident{Name: faceDecl.Name.Name}),
					Type: &dst.FuncType{
						Params:  faceDecl.Type.Params,
						Results: faceDecl.Type.Results,
					},
				},
				)
			}
		}
		return true
	})

	if _, ok := apiMap["I"+forStruct]; ok {
		fmt.Printf("已经存在接口 %s 执行终止操作!", "I"+forStruct)
		return
	}
	dstFileList := &dst.FieldList{
		List: dstMethods,
	}

	apiFace := &dst.GenDecl{
		Tok: token.TYPE,
		Specs: append(make([]dst.Spec, 0), &dst.TypeSpec{
			Name: &dst.Ident{Name: "I" + forStruct},
			Type: &dst.InterfaceType{
				Methods: dstFileList,
			},
		}),
	}

	newDecl := make([]dst.Decl, 0)
	for _, v := range f.Decls {
		if _, ok := v.(*dst.GenDecl); ok {
			if v.(*dst.GenDecl).Tok == token.TYPE {
				newDecl = append(newDecl, v, apiFace)
				continue
			}
		}
		newDecl = append(newDecl, v)

	}
	f.Decls = newDecl
	f.Decls = newDecl
	tmpeOut := "/tmp/face.go"
	os.Remove(tmpeOut)
	ret, _ := os.OpenFile(tmpeOut, os.O_WRONLY|os.O_CREATE, 0666)
	if err := decorator.Fprint(ret, f); err != nil {
		panic(err)
	}
	// dst.Print(dsTree)

}
