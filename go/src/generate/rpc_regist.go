package main

import (
	"bytes"
	"common"
	"os"
	"regexp"
	"text/template"
)

const (
	K_RegistOutDir   = K_OutDir
	K_RegistFileName = "generate_rpc.go"
)

type Func struct {
	Pack string //package
	Name string
}
type RpcInfo struct {
	Svr           string          //服务器名
	Moudles       map[string]bool //package
	TcpRpc        []Func
	HttpRpc       []Func
	HttpPlayerRpc []Func
	HttpHandle    []Func
}

func generatRpcRegist(svr string) *RpcInfo {
	names, _ := common.WalkDir(K_SvrDir+svr, ".go")
	pinfo := &RpcInfo{Svr: svr, Moudles: make(map[string]bool)}
	for _, v := range names {
		moudle := "" //package name
		common.ReadLine(v, func(line string) {
			fname := "" //func name
			if fname = getPackage(line); fname != "" {
				moudle = fname
				fname = ""
			} else if fname = getTcpRpc(line); fname != "" {
				pinfo.TcpRpc = append(pinfo.TcpRpc, Func{moudle, fname})
			} else if fname = getHttpRpc(line); fname != "" {
				pinfo.HttpRpc = append(pinfo.HttpRpc, Func{moudle, fname})
			} else if fname = getHttpPlayerRpc(line); fname != "" {
				pinfo.HttpPlayerRpc = append(pinfo.HttpPlayerRpc, Func{moudle, fname})
			} else if fname = getHttpHandle(line); fname != "" {
				pinfo.HttpHandle = append(pinfo.HttpHandle, Func{moudle, fname})
			}
			if moudle != "" && fname != "" {
				pinfo.Moudles[moudle] = true
			}
		})
	}
	pinfo.makeFile(svr)
	return pinfo
}

// -------------------------------------
// -- 提取 package、RpcFunc
func getPackage(s string) string {
	if ok, _ := regexp.MatchString(`^package \w+$`, s); ok {
		reg := regexp.MustCompile(`\w+`)
		return reg.FindAllString(s, -1)[1]
	}
	return ""
}
func getTcpRpc(s string) string {
	if ok, _ := regexp.MatchString(`^func Rpc_\w+\(\w+, \w+ \*common.NetPack, \w+ \*tcp.TCPConn\) \{$`, s); ok {
		reg := regexp.MustCompile(`Rpc_\w+`)
		return reg.FindAllString(s, -1)[0]
	}
	return ""
}
func getHttpRpc(s string) string {
	if ok, _ := regexp.MatchString(`^func Rpc_\w+\(\w+, \w+ \*common.NetPack\) \{$`, s); ok {
		reg := regexp.MustCompile(`Rpc_\w+`)
		return reg.FindAllString(s, -1)[0]
	}
	return ""
}
func getHttpPlayerRpc(s string) string {
	if ok, _ := regexp.MatchString(`^func Rpc_\w+\(\w+, \w+ \*common.NetPack, \w+ interface\{\}\) \{$`, s); ok {
		reg := regexp.MustCompile(`Rpc_\w+`)
		return reg.FindAllString(s, -1)[0]
	}
	return ""
}
func getHttpHandle(s string) string {
	if ok, _ := regexp.MatchString(`^func Http_\w+\(\w+ http.ResponseWriter, \w+ \*http.Request\) \{$`, s); ok {
		reg := regexp.MustCompile(`Http_\w+`)
		return reg.FindAllString(s, -1)[0][5:]
	}
	return ""
}

// -------------------------------------
// -- 填充模板
const codeRegistTemplate = `
// Generated by GoServer/src/generat
// Don't edit !
package rpc
import (
	"netConfig"
	"generate_out/rpc/enum"
	{{range $k, $_ := .Moudles}}
	{{if eq $k "logic"}}
	"svr_{{$.Svr}}/{{$k}}"{{else}}
	"svr_{{$.Svr}}/logic/{{$k}}"{{end}}{{end}}
)
func init() {
	netConfig.RegTcpRpc(map[uint16]netConfig.TcpRpc{
		{{range .TcpRpc}}
		enum.{{.Name}}: {{.Pack}}.{{.Name}},{{end}}
	})
	netConfig.RegHttpRpc(map[uint16]netConfig.HttpRpc{
		{{range .HttpRpc}}
		enum.{{.Name}}: {{.Pack}}.{{.Name}},{{end}}
	})
	netConfig.RegHttpPlayerRpc(map[uint16]netConfig.HttpPlayerRpc{
		{{range .HttpPlayerRpc}}
		enum.{{.Name}}: {{.Pack}}.{{.Name}},{{end}}
	})
	netConfig.RegHttpHandler(map[string]netConfig.HttpHandle{
		{{range .HttpHandle}}
		"{{.Name}}": {{.Pack}}.Http_{{.Name}},{{end}}
	})
}
`

func (self *RpcInfo) makeFile(svr string) {
	filename := K_RegistFileName
	tpl, err := template.New(filename).Parse(codeRegistTemplate)
	if err != nil {
		panic(err.Error())
		return
	}
	var bf bytes.Buffer
	if err = tpl.Execute(&bf, self); err != nil {
		panic(err.Error())
		return
	}
	if err := os.MkdirAll(K_RegistOutDir+svr, 0777); err != nil {
		panic(err.Error())
		return
	}
	f, err := os.OpenFile(K_RegistOutDir+svr+"/"+filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err.Error())
		return
	}
	defer f.Close()
	f.Write(bf.Bytes())
}
