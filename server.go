package main

import (
	"bytes"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/moby/moby/client"
	"golang.org/x/net/context"
	"io"
	"net/http"
	// "os"
	// "reflect"
)

// コード実行用のJSONパラメータ
type ExecParams struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Cmd      string `json:"cmd"`
	Input    string `json:"input"`
	WorkDir  string `json:"workDir"`
}

func exec(c echo.Context) error {
	params := new(ExecParams)

	if err := c.Bind(params); err != nil {
		panic(err)
	}

	fmt.Println("================")
	fmt.Println(params)
	fmt.Println(params.Language)
	fmt.Println(params.Code)
	fmt.Println(params.Cmd)
	fmt.Println("================")

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      "codecandy_compiler_default",
		Cmd:        []string{"pwd"},
		Tty:        true,
		WorkingDir: "/workspace",
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

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
	fmt.Println(newStr)
	fmt.Println("===============")

	// err = cli.ContainerRemove(ctx, "id", types.ContainerRemoveOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	jsonMap := map[string]string{
		"status": "Active",
		"exec":   newStr,
	}

	return c.JSON(http.StatusOK, jsonMap)
}

// APIのステータスを返却
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
