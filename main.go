package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func ParseJsonFile(dest interface{}, file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, dest)
	return err
}

var tmpl, _ = template.New("main").Parse(TMPL_MAIN)

func showCmdListPage(w http.ResponseWriter, req *http.Request) {
	tmpl.Execute(w, _config.Cmds)
}

func showCmdResultInitPage(w http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id")
	html := strings.Replace(_html, "{id}", id, -1)
	io.WriteString(w, html)
}
func FormatPath(path string) string {
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.TrimRight(path, "/")
	return path
}
func GetPath(file string) string {
	file = FormatPath(file)
	pos := strings.LastIndex(file, "/")
	return file[0:pos]
}
func WriteStringFile(file string, str string) (written int, err error) {
	//确保创建目标目录
	//CreateDir(file)
	dst, err := os.Create(file)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.WriteString(dst, str)
}
func exec_cmd(id int, w *websocket.Conn) {
	cmdCfg := &_config.Cmds[id]
	if cmdCfg.Running {
		websocket.Message.Send(w, "The script is running, please waitting .......")
		return
	}
	cmdCfg.Running = true
	strCmd := cmdCfg.Script
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		content, err := ioutil.ReadFile(cmdCfg.Script)
		if err != nil {
			websocket.Message.Send(w, err.Error())
			return
		}
		strCmd = cmdCfg.Script + ".tmp" + strconv.Itoa(id) + ".bat"
		WriteStringFile(strCmd, "@echo off \r\n chcp 65001 \r\n"+string(content))
		defer os.Remove(strCmd)
	}
	cmd = exec.Command(strCmd)
	cmd.Env = os.Environ()
	cmd.Stdout = w

	path := GetPath(cmdCfg.Script)
	cmd.Dir = path

	err := cmd.Start()
	if err != nil {
		websocket.Message.Send(w, err.Error())
		return
	}

	cmd.Wait()
	cmdCfg.Running = false
	cmdCfg.LastRunTime = time.Now()
	websocket.Message.Send(w, "\n---------------------\nRUN OVER")
}

func execAndRefreshCmdResult(ws *websocket.Conn) {
	req := ws.Request()
	id, _ := strconv.Atoi(req.FormValue("id"))
	if id >= len(_config.Cmds) {
		websocket.Message.Send(ws, "Invalid Command.")
		return
	}

	//ws.SetWriteDeadline(time.Now().Add(20 * time.Second))
	exec_cmd(id, ws)
}

type Cmd struct {
	Text        string
	Script      string
	Url         string
	Running     bool
	LastRunTime time.Time
}

type Config struct {
	Port int
	Cmds []Cmd
}

var _html string
var _config Config
var port int

func main() {
	flag.Parse()
	ParseJsonFile(&_config, "config.json")
	port = _config.Port
	_html = strings.Replace(HTML_EXEC, "{port}", strconv.Itoa(port), -1)

	http.HandleFunc("/", showCmdListPage)
	http.HandleFunc("/cmd", showCmdResultInitPage)
	http.Handle("/exec", websocket.Handler(execAndRefreshCmdResult))

	fmt.Printf("http://localhost:%d/\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

const HTML_EXEC = `
<html>
<head>
<script type="text/javascript">
var path;
var ws;
function init() {
   console.log("init");
   if (ws != null) {
     ws.close();
     ws = null;
   }
   var div = document.getElementById("msg");
   div.innerText =  "\n" + div.innerText;
   ws = new WebSocket("ws://localhost:{port}" + "/exec?id={id}");
   ws.binaryType ="string";
   ws.onopen = function () {
    //div.innerText = "opened\n" + div.innerText;
	//ws.send("ok");
   };
   ws.onmessage = function (e) {
      div.innerText = div.innerText + e.data + "\n";
   };
   ws.onclose = function (e) {
     // div.innerText = div.innerText + "closed";
   };
   //div.innerText = "init\n" + div.innerText;
};
</script>
<body onLoad="init();"/>
<div id="msg"></div>
</html>
`

const TMPL_MAIN = `
<html>
<head>
</head>
<body>
<table border="0" cellspacing="8">
	<thead><tr><th>Name</th><th></th><th>Last run time</th></tr></thead>
	{{with .}}
	{{range $k, $v := .}}
	<tr>
		<td><a href="/cmd?id={{$k}}">{{$v.Text}}</td>
		<td><a href="{{$v.Url}}">Download</td>
		{{with $v.LastRunTime}}
		<td>{{.}}</td>
		{{end}}
	</tr>
	{{end}}
	{{end}}
</table>
</body>
</html>
`
