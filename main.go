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
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var port int = *flag.Int("p", 23456, "Port to listen.")

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
func exec_cmd(id int, w io.Writer) {
	cmdCfg := &_config.Cmds[id]
	if cmdCfg.Running {
		w.Write([]byte("The script is running, please waitting ......."))
		return
	}
	cmdCfg.Running = true
	cmd := exec.Command(cmdCfg.Script)
	cmd.Stdout = w
	path := GetPath(cmdCfg.Script)
	cmd.Dir = path

	err := cmd.Start()
	if err != nil {
		fmt.Println("Exec Error:", err.Error())
	}

	cmd.Wait()
	cmdCfg.Running = false
	cmdCfg.LastRunTime = time.Now()
	w.Write([]byte("\n---------------------\nRUN OVER"))
}

func execAndRefreshCmdResult(ws *websocket.Conn) {
	req := ws.Request()
	id, _ := strconv.Atoi(req.FormValue("id"))
	if id >= len(_config.Cmds) {
		websocket.Message.Send(ws, "Invalid Command.")
		return
	}
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

func main() {
	flag.Parse()
	ParseJsonFile(&_config, "config.json")
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
   ws.onopen = function () {
    //div.innerText = "opened\n" + div.innerText;
	//ws.send("ok");
   };
   ws.onmessage = function (e) {
      div.innerText = div.innerText + e.data;
   };
   ws.onclose = function (e) {
      //div.innerText = "closed\n" + div.innerText;
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
