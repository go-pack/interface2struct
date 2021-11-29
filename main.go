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

type methods = map[*dst.FuncType]bool

var (
	interfaceName = flag.String("s", "IIndexService", "interface名称")
	isOverWrite   = flag.Bool("m", false, "是否覆盖")
	structName    = flag.String("name", *interfaceName+"Impl", "structName")
	output        = flag.String("o", "/tmp/xx.go", "输出位置")
	apiMap        = make(map[string]bool)

	structMap          = make(map[string]methods)
	hasInterfaceInFile = false

	proxyTargetStr = "IndexService"
	proxyTarget    = &proxyTargetStr
)

//go:generate go version
func main() {
	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		// return
	}
	if *interfaceName == "" {
		fmt.Printf("请输入接口名称!")
	}
	wd, _ := os.Getwd()
	file := os.Getenv("GOFILE")
	pack := os.Getenv("GOPACKAGE")

	path := wd + string(os.PathSeparator) + file

	path = "/Users/chen/IdeaProjects/smm-go/internal/services/indexService.go"

	fmt.Printf("wd %s file %s pack %s path %s \r\n", wd, file, pack, path)
	fset := token.NewFileSet()
	f, err := decorator.ParseFile(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}
	interfaceImport := make([]dst.Spec, 0)

	// dstMethods := make([]*dst.Field, 0)
	interfaceDecl := &dst.InterfaceType{}
	interfaceDecl = nil
	dst.Inspect(f, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.GenDecl:
			if x.Tok == token.TYPE {
				genDecl := dst.Clone(x)
				if val, ok := x.Specs[0].(*dst.TypeSpec); ok {
					if _, ok := val.Type.(*dst.InterfaceType); ok {
						if val.Name.Name == *interfaceName {
							interfaceDecl = genDecl.(*dst.GenDecl).Specs[0].(*dst.TypeSpec).Type.(*dst.InterfaceType)
						}
					}
				}
			}
		}
		return true
	})

	if interfaceDecl == nil {
		fmt.Printf("接口 %s 不存在,执行终止操作!", *interfaceName)
		return
	}
	importMap := make(map[string]bool)
	dst.Inspect(interfaceDecl, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.SelectorExpr:
			importMap[x.X.(*dst.Ident).Name] = true
		}
		return true
	})

	dst.Inspect(f, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.ImportSpec:
			if x.Name != nil {
				if _, ok := importMap[x.Name.Name]; ok {
					interfaceImport = append(interfaceImport, x)
				}
			} else {
				paths := strings.Split(x.Path.Value, "/")
				lastPath := paths[len(paths)-1]
				lastPath = strings.Trim(lastPath, "\"")
				if _, ok := importMap[lastPath]; ok {
					interfaceImport = append(interfaceImport, x)
				}
			}

			fmt.Printf("d %+v", x)

		}
		return true
	})
	fmt.Printf("d %+v", interfaceImport)

	typeSpec := &dst.TypeSpec{}
	typeSpec.Name = &dst.Ident{
		Name: *structName,
		Obj: &dst.Object{
			Kind: dst.Typ,
			Name: *structName,
		},
	}
	typeSpec.Type = &dst.StructType{
		Fields: &dst.FieldList{
			List: append(make([]*dst.Field, 0), &dst.Field{
				Type: &dst.Ident{
					Name: "Instance",
					Obj: &dst.Object{
						Kind: dst.Typ,
					},
				},
				Names: append(make([]*dst.Ident, 0), &dst.Ident{
					Name: "instance",
					Obj: &dst.Object{
						Kind: dst.Var,
						Name: "instance",
					},
				}),
			}),
		},
	}
	structFuncDecl := &dst.FuncDecl{
		Name: &dst.Ident{
			Name: "New" + *structName,
			Obj: &dst.Object{
				Name: "New" + *structName,
				Kind: dst.Fun,
			},
		},
		Type: &dst.FuncType{
			Params: &dst.FieldList{List: append(make([]*dst.Field, 0), &dst.Field{
				Names: append(make([]*dst.Ident, 0), &dst.Ident{
					Name: "src",
				}),
				Type: &dst.Ident{Name: *proxyTarget},
			})},
			Results: &dst.FieldList{List: append(make([]*dst.Field, 0), &dst.Field{
				Names: append(make([]*dst.Ident, 0)),
				Type: &dst.Ident{Name: *proxyTarget, Obj: &dst.Object{
					Name: *proxyTarget,
					Kind: dst.Typ,
				}},
			})},
		},
		Body: &dst.BlockStmt{
			List: append(make([]dst.Stmt, 0), &dst.ReturnStmt{
				Results: append(make([]dst.Expr, 0), &dst.UnaryExpr{
					Op: token.AND,
					X: &dst.CompositeLit{
						Type: &dst.Ident{
							Name: *structName,
							Obj: &dst.Object{
								Name: *structName,
								Kind: dst.Typ,
							},
						},
						Elts: append(make([]dst.Expr, 0), &dst.KeyValueExpr{
							Key: &dst.Ident{Name: "proxy"},
							Value: &dst.Ident{Name: "src", Obj: &dst.Object{
								Name: "src",
								Kind: dst.Var,
							}},
						}),
					},
				}),
			}),
		},
	}
	funcDecls := append(make([]dst.Decl, 0),
		&dst.GenDecl{
			Tok:   token.IMPORT,
			Specs: append(make([]dst.Spec, 0), interfaceImport...),
		},
		&dst.GenDecl{
			Tok:   token.TYPE,
			Specs: append(make([]dst.Spec, 0), typeSpec),
		},
		structFuncDecl,
	)
	for _, x := range interfaceDecl.Methods.List {
		fmt.Printf(" f %+v", x)
		params := make([]dst.Expr, 0)
		isNameReturn := false
		funcType := x.Type.(*dst.FuncType)
		if funcType.Results != nil {
			for _, result := range funcType.Results.List {
				if len(result.Names) > 0 {
					isNameReturn = true
					break
				}
			}
		}
		for _, v := range funcType.Params.List {
			params = append(params, &dst.Ident{
				Name: v.Names[0].Name,
				Obj: &dst.Object{
					Name: v.Names[0].Name,
					Kind: dst.Var,
				},
			})
		}
		argsCount := 1
		returns := make([]dst.Expr, 0)
		assings := make([]dst.Expr, 0)
		var tok token.Token

		for _, result := range funcType.Results.List {
			if isNameReturn {
				tok = token.ASSIGN
				returns = append(returns, &dst.Ident{Name: result.Names[0].Name})
				assings = append(assings, &dst.Ident{Name: result.Names[0].Name})
			} else {
				tok = token.DEFINE
				returns = append(returns, &dst.Ident{Name: fmt.Sprintf("args%d", argsCount)})
				assings = append(assings, &dst.Ident{Name: fmt.Sprintf("args%d", argsCount)})
				argsCount++
			}
		}

		funcDecls = append(funcDecls, &dst.FuncDecl{
			Recv: &dst.FieldList{
				List: append(make([]*dst.Field, 0), &dst.Field{
					Names: append(make([]*dst.Ident, 0), &dst.Ident{
						Name: "t",
						Obj: &dst.Object{
							Name: "t",
							Kind: dst.Var,
						},
					}),
					Type: &dst.StarExpr{
						X: &dst.Ident{
							Name: *structName,
							Obj: &dst.Object{
								Name: *structName,
								Kind: dst.Typ,
							},
						},
					},
				}),
			},
			Name: dst.NewIdent(x.Names[0].Name),
			Type: x.Type.(*dst.FuncType),
			Body: &dst.BlockStmt{
				List: append(make([]dst.Stmt, 0),
					&dst.AssignStmt{
						Lhs: assings,
						Tok: tok,
						Rhs: append(make([]dst.Expr, 0), &dst.CallExpr{
							Fun: &dst.SelectorExpr{
								X:   &dst.Ident{Name: "proxy"},
								Sel: dst.NewIdent(x.Names[0].Name),
							},
							Args: params,
						}),
					}, &dst.ReturnStmt{
						Results: returns,
					},
				),
			},
		})
	}
	dsTree := dst.File{
		Name:  &dst.Ident{Name: "Proxy" + *proxyTarget},
		Decls: funcDecls,
	}
	fmt.Printf("%+v", funcDecls)

	// tempOut := *output
	// if *isOverWrite {
	// 	tempOut = path
	// }
	// ret, _ := os.OpenFile(tempOut, os.O_WRONLY|os.O_CREATE, 0666)
	// if err := decorator.Fprint(ret, f); err != nil {
	// 	panic(err)
	// }
	// dst.Print(dsTree)
	if err := decorator.Print(&dsTree); err != nil {
		panic(err)
	}
}
