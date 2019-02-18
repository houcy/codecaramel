package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	// "log"
	// "reflect"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/moby/moby/client"
	"golang.org/x/net/context"
)

// コード実行用のJSONパラメータ
type ExecParams struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Input    string `json:"input"`
}

type Cmd struct {
	Query string `json:"query"`
}

// < > &をエスケープしないようにする関数
func JSONMarshal(v interface{}, safeEncoding bool) ([]byte, error) {
	b, err := json.Marshal(v)
	if safeEncoding {
		b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
		b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
		b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	}
	return b, err
}

// コンテナイメージ名を返却する
func imgName(language string) string {
	switch language {
	case "Java", "Scala", "PHP":
		return "codecaramel_compiler_jvm_php"
	case "Swift":
		return "codecaramel_compiler_swift"
	default:
		return "codecaramel_compiler_default"
	}
}

// ファイル名を返却する
func getFileName(language string) string {
	switch language {
	case "Gcc", "Clang":
		return "main.c"
	case "Ruby":
		return "main.rb"
	case "Python3":
		return "main.py"
	case "Golang":
		return "main.go"
	case "Nodejs":
		return "main.js"
	case "Java":
		return "Main.java"
	case "Scala":
		return "Main.scala"
	case "Swift":
		return "main.swift"
	case "CPP":
		return "main.cpp"
	case "PHP":
		return "main.php"
	case "Perl":
		return "main.pl"
	case "Bash":
		return "main.sh"
	case "Lua":
		return "main.lua"
	case "Haskell":
		return "main.hs"
	}
	return "main"
}

func getCmd(language string) string {
	switch language {
	case "Gcc", "Clang":
		return "gcc -o main main.c && ./main"
	case "Ruby":
		return "ruby main.rb"
	case "Python3":
		return "python main.py"
	case "Golang":
		return "go run main.go"
	case "Nodejs":
		return "nodejs main.js"
	case "Java":
		return "javac Main.java && java Main"
	case "Scala":
		return "scalac Main.scala && scala Main"
	case "Swift":
		return "swift main.swift"
	case "CPP":
		return "g++ -o main main.cpp && ./main"
	case "PHP":
		return "php main.php"
	case "Perl":
		return "perl main.pl"
	case "Bash":
		return "bash main.sh"
	case "Lua":
		return "lua 5.3 main.lua"
	case "Haskell":
		return "runghc main.hs"
	}
	return "No cmd"
}

func compilerWorker(params *ExecParams, cli *client.Client, ctx context.Context, resp container.ContainerCreateCreatedBody, workDir string) <-chan string {
	receiver := make(chan string)

	go func() {
		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			panic(err)
		}

		// 実行が終わるまで待機
		cli.ContainerWait(ctx, resp.ID)

		out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})

		if err != nil {
			panic(err)
		}

		buf := new(bytes.Buffer)
		io.Copy(buf, out)
		newStr := buf.String()

		receiver <- newStr
		close(receiver)
	}()
	return receiver
}

/**
* POST: /api/container/exec
* 提出されたコードを実行する
**/
func exec(c echo.Context) error {
	// リクエストされたパラメータを格納
	params := new(ExecParams)
	if err := c.Bind(params); err != nil {
		panic(err)
	}

	// データの事前準備
	now := time.Now().Unix()
	workDir := strconv.FormatInt(now, 10)

	// フォルダの作成
	if err := os.Mkdir("/tmp/"+workDir, 0777); err != nil {
		fmt.Println(err)
	}

	// ファイルの作成
	code := []byte(params.Code)
	ioutil.WriteFile("/tmp/"+workDir+"/"+getFileName(params.Language), code, os.ModePerm)

	// 標準入力用のファイル作成
	input := []byte(params.Input)
	ioutil.WriteFile("/tmp/"+workDir+"/input", input, os.ModePerm)

	// 実行用スクリプトの作成
	cmd := []byte("#!/bin/bash\ncat input | " + getCmd(params.Language)) // 実行コマンドの生成
	ioutil.WriteFile("/tmp/"+workDir+"/exec.sh", cmd, os.ModePerm)

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      imgName(params.Language),
		Cmd:        strings.Split("bash exec.sh", " "), // strings.Split("ls", " "),
		Tty:        true,
		WorkingDir: "/workspace",
	}, &container.HostConfig{
		Binds: []string{"/tmp/" + workDir + ":/workspace"},
	}, nil, "")

	receiver := compilerWorker(params, cli, ctx, resp, workDir)
	timeout := 30 * time.Second

	select {
	case receive := <-receiver:
		jsonMap := map[string]string{
			"status": "Active",
			"result": receive,
		}

		err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
		if err != nil {
			panic(err)
		}

		if err := os.RemoveAll("/tmp/" + workDir); err != nil {
			fmt.Println(err)
		}

		return c.JSON(http.StatusOK, jsonMap)
	case <-time.After(timeout):
		err = cli.ContainerStop(ctx, resp.ID, &timeout)
		if err != nil {
			panic(err)
		}
		if err := os.RemoveAll("/tmp/" + workDir); err != nil {
			fmt.Println(err)
		}
		err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
		if err != nil {
			panic(err)
		}
		jsonMap := map[string]string{
			"status": "Timeout",
			"exec":   "",
		}
		return c.JSON(http.StatusOK, jsonMap)
	}
}

/**
* GET: /api/compiler/status
* APIのステータスを返却
**/
func status(c echo.Context) error {
	host, _ := os.Hostname()
	jsonMap := map[string]string{
		"status":   "Active",
		"hostname": host,
	}
	return c.JSON(http.StatusOK, jsonMap)
}

func main() {
	e := echo.New()

	// == middleware ==
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// ================

	// == routing ==
	e.GET("/api/compiler/status", status)
	e.POST("/api/compiler/exec", exec)
	// =============

	e.Start(":5000")
}
