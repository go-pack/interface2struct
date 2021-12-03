package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/token"
	"os"
	sysPath "path"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func FileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

type methods = map[*dst.FuncType]bool

var (
	interfaceName = flag.String("s", "IIndexService", "interface名称")
	isOverWrite   = flag.Bool("m", false, "是否覆盖")
	structName    = flag.String("name", *interfaceName+"Impl", "structName")
	output        = flag.String("o", "/tmp/xx.go", "输出位置")
	proxyTarget   = flag.String("p", "", "要代理的结构")
	toPackageName = flag.String("t", "", "生成的包名称")
	apiMap        = make(map[string]bool)

	structMap          = make(map[string]methods)
	oldMethodMap       = make(map[string]*dst.FuncDecl)
	interfaceMethodMap = make(map[string]*dst.FuncDecl)
	hasInterfaceInFile = false
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
	if os.Getenv("USE_WD") != "" {
		wd = os.Getenv("USE_WD")
	}
	buf, err := os.ReadFile(sysPath.Join(wd, "go.mod"))
	if err != nil {
		fmt.Printf("gomod read %s \r\n", err)
		return
	}
	reader := bufio.NewReader(strings.NewReader(string(buf)))
	line, _, err := reader.ReadLine()
	if err != nil {
		fmt.Printf("ReadLine  %s \r\n", err)
		return
	}
	if !strings.Contains(string(line), "module") {
		fmt.Printf("goMode module missing \r\n")
		return
	}
	mod := strings.Split(string(line), " ")[1]
	file := os.Getenv("GOFILE")
	pack := os.Getenv("GOPACKAGE")

	path := wd + string(os.PathSeparator) + file

	tempOut := *output
	if *isOverWrite {
		tempOut = path
	} else {
		nameStr := *structName
		name := strings.ToLower(nameStr[:1]) + nameStr[1:]
		tempOut = wd + string(os.PathSeparator) + tempOut + string(os.PathSeparator) + name + ".go"
	}
	if FileExist(tempOut) {
		buf, err := os.ReadFile(tempOut)
		if len(buf) > 0 && err == nil {
			fset := token.NewFileSet()
			olfTree, err := decorator.ParseFile(fset, tempOut, nil, 0)
			if err != nil {
				fmt.Println("存在就文件 " + tempOut + " 读取错误 " + err.Error())
				return
			}
			dst.Inspect(olfTree, func(n dst.Node) bool {
				switch x := n.(type) {
				case *dst.FuncDecl:
					oldMethodMap[x.Name.Name] = x
				}
				return true
			})
		}

	}
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
	//添加接口import
	// interfaceImport = append(interfaceImport, &dst.ImportSpec{
	// 	Name: nil,
	// 	Path: &dst.BasicLit{
	// 		Value: "\"" + sysPath.Join(mod, filepath.Dir(file)) + "\"",
	// 	},
	// })
	fmt.Printf("d %+v mod %s", interfaceImport, mod)

	typeSpec := &dst.TypeSpec{}
	typeSpec.Name = &dst.Ident{
		Name: *structName,
		Obj: &dst.Object{
			Kind: dst.Typ,
			Name: *structName,
		},
	}
	//结构代理类的属性字段
	if *proxyTarget != "" {
		typeSpec.Type = &dst.StructType{
			Fields: &dst.FieldList{
				List: append(make([]*dst.Field, 0), &dst.Field{
					Type: &dst.Ident{
						Name: *proxyTarget,
						Obj: &dst.Object{
							Kind: dst.Typ,
						},
					},
					Names: append(make([]*dst.Ident, 0), &dst.Ident{
						Name: "proxy",
						Obj: &dst.Object{
							Kind: dst.Var,
							Name: "proxy",
						},
					}),
				}),
			},
		}
	} else {
		typeSpec.Type = &dst.StructType{
			Fields: &dst.FieldList{
				List: append(make([]*dst.Field, 0), &dst.Field{
					Type: &dst.Ident{
						Name: *proxyTarget,
						Obj: &dst.Object{
							Kind: dst.Typ,
						},
					},
					Names: append(make([]*dst.Ident, 0)),
				}),
			},
		}
	}
	//构造参数
	params := append(make([]*dst.Field, 0))
	if *proxyTarget != "" {
		params = append(params, &dst.Field{
			Names: append(make([]*dst.Ident, 0), &dst.Ident{
				Name: "src",
			}),
			Type: &dst.Ident{Name: *proxyTarget},
		})
	}
	//构造内容
	body := make([]dst.Expr, 0)
	if *proxyTarget != "" {
		body = append(body, &dst.UnaryExpr{
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
		})
	} else {
		body = append(body, &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: *structName,
					Obj: &dst.Object{
						Name: *structName,
						Kind: dst.Typ,
					},
				},
			},
		})
	}
	//构造函数
	newFuncName := "New" + *structName

	structFuncDecl := &dst.FuncDecl{
		Name: &dst.Ident{
			Name: newFuncName,
			Obj: &dst.Object{
				Name: newFuncName,
				Kind: dst.Fun,
			},
		},
		Type: &dst.FuncType{
			Params: &dst.FieldList{List: params},
			Results: &dst.FieldList{List: append(make([]*dst.Field, 0), &dst.Field{
				Names: append(make([]*dst.Ident, 0)),
				Type: &dst.StarExpr{X: &dst.Ident{
					Name: *structName, Obj: &dst.Object{
						Name: *structName,
						Kind: dst.Typ,
					},
				}},
			})},
		},
		Body: &dst.BlockStmt{
			List: append(make([]dst.Stmt, 0), &dst.ReturnStmt{
				Results: body,
			}),
		},
	}
	if val, ok := oldMethodMap[newFuncName]; ok {
		structFuncDecl = val
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
		if valFunc, ok := oldMethodMap[x.Names[0].Name]; ok {
			funcDecls = append(funcDecls, valFunc)
			interfaceMethodMap[x.Names[0].Name] = valFunc
		} else {
			body := make([]dst.Stmt, 0)
			if *proxyTarget != "" {
				body = append(body, &dst.AssignStmt{
					Lhs: assings,
					Tok: tok,
					Rhs: append(make([]dst.Expr, 0), &dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X: &dst.SelectorExpr{
								X:   &dst.Ident{Name: "t"},
								Sel: dst.NewIdent("proxy"),
							},
							Sel: dst.NewIdent(x.Names[0].Name),
						},
						Args: params,
					}),
				}, &dst.ReturnStmt{
					Results: returns,
				})
			} else {
				body = append(make([]dst.Stmt, 0), &dst.ExprStmt{
					X: &dst.CallExpr{
						Fun: &dst.Ident{Name: "panic"},
						Args: append(make([]dst.Expr, 0), &dst.BasicLit{
							Kind:  token.STRING,
							Value: "\"need implement\"",
						}),
					},
				})
			}
			varFunc := &dst.FuncDecl{
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
					List: body,
				},
			}
			interfaceMethodMap[x.Names[0].Name] = valFunc
			funcDecls = append(funcDecls, varFunc)
		}
	}
	//如果新结构代码有新建内部方法,则恢复新方法
	for k, v := range oldMethodMap {
		if _, ok := interfaceMethodMap[k]; !ok && k != newFuncName {
			funcDecls = append(funcDecls, v)
		}
	}
	dsTree := dst.File{
		Decls: funcDecls,
	}
	if *proxyTarget != "" {
		dsTree.Name = &dst.Ident{Name: "Proxy" + *proxyTarget}
	} else {
		dsTree.Name = &dst.Ident{Name: *toPackageName}
	}
	//	os.Remove(tempOut)
	ret, _ := os.OpenFile(tempOut, os.O_WRONLY|os.O_CREATE, 0666)
	if err := decorator.Fprint(ret, &dsTree); err != nil {
		panic(err)
	}
	// dst.Print(dsTree)
	if err := decorator.Print(&dsTree); err != nil {
		panic(err)
	}
}
