package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func ParseJsonFile(dest interface{}, file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, dest)
	return err
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
