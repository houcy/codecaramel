package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/moby/moby/client"
	"golang.org/x/net/context"
	// "reflect"
)

// コード実行用のJSONパラメータ
type ExecParams struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Input    string `json:"input"`
}

// コンテナイメージ名を返却する
func imgName(language string) string {
	switch language {
	case "Java", "Scala", "PHP":
		return "codecandy_compiler_jvm_php"
	case "Swift":
		return "codecandy_compiler_swift"
	default:
		return "codecandy_compiler_default"
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

	// workDir名をUnix時間から作成
	now := time.Now().Unix()
	workDir := strconv.FormatInt(now, 10)

	fmt.Println("================")
	fmt.Println(params)
	fmt.Println(params.Language)
	fmt.Println(params.Code)
	fmt.Println(imgName(params.Language))
	fmt.Println(now)
	fmt.Println(workDir)
	fmt.Println("================")

	// データの事前準備
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

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	cmd := "bash -c cd /workspace && /usr/bin/time -q -f \"%e\" -o /workspace/time.txt timeout 30 " + getCmd(params.Language) + " < input"

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      imgName(params.Language),
		Cmd:        strings.Split(cmd, " "), // strings.Split("pwd", " "),
		Tty:        true,
		WorkingDir: "/workspace",
	}, &container.HostConfig{
		Binds: []string{"/tmp/" + workDir + ":/workspace"},
	}, nil, "")

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	if _, err = cli.ContainerWait(ctx, resp.ID); err != nil {
		panic(err)
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})

	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	io.Copy(buf, out)
	newStr := buf.String()
	fmt.Println("===============")
	fmt.Println(cmd)
	fmt.Println(newStr)
	fmt.Println("===============")

	// err = cli.ContainerRemove(ctx, "id", types.ContainerRemoveOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	if err := os.RemoveAll("/tmp/" + workDir); err != nil {
		fmt.Println(err)
	}

	jsonMap := map[string]string{
		"status": "Active",
		"exec":   newStr,
	}

	return c.JSON(http.StatusOK, jsonMap)
}

/**
* GET: /api/compiler/status
* APIのステータスを返却
**/
func status(c echo.Context) error {
	fmt.Println("/api/compiler/exec")

	jsonMap := map[string]string{
		"status": "Active",
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

	e.Start(":4567")
}
