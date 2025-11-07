package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "golang.org/x/image/webp"
)

type ResponseJson struct {
    Code int `json:"code" xml:"code"`
    Url string `json:"url" xml:"url"`
    Type string `json:"type" xml:"type"`
    Message string `json:"message" xml:"message"`
}

var host string = ""
var imagePath string = ""

func validateImage(file multipart.File) (string, error) {
    _, err := file.Seek(0, io.SeekStart)
    if err != nil {
        return "", err
    }
    _, imgType, err := image.Decode(file)
    if err != nil {
        return "", err
    }
    _, err = file.Seek(0, io.SeekStart)
    return imgType, nil
}

func saveUploadFile(file *multipart.FileHeader, dst string) error {
    fmt.Println("dst:", dst)
    src, err := file.Open()
    if err != nil {
        fmt.Println("file open error")
        return err
    }
    defer src.Close()

    out, err := os.Create(dst)
    if err != nil {
        fmt.Println("create dst error")
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, src)
    if err != nil {
        fmt.Println("copy error")
    }
    return err
}

func upload(c echo.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.JSON(http.StatusBadRequest, ResponseJson{Code: 400, Message: "参数错误"})
    }
    src, err := file.Open()
    if err != nil {
        return c.JSON(http.StatusBadRequest, ResponseJson{Code: 400, Message: "参数错误"})
    }
    defer src.Close()

    imgType, err := validateImage(src)
    if err != nil {
        return c.JSON(http.StatusBadRequest, ResponseJson{Code: 400, Message: "上传文件不是图片"})
    }
    suffix := path.Ext(file.Filename)
    uuid, err := uuid.NewV7()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ResponseJson{Code: 500, Message: "系统错误"})
    }
    filePath := time.Unix(uuid.Time().UnixTime()).Format("200601")
    filePath = imagePath + filePath
    _, err = os.Stat(filePath)
    if err != nil {
        err = os.MkdirAll(filePath, os.ModePerm)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, ResponseJson{Code: 500, Message: "系统错误"})
        }
    }
    filename := fmt.Sprintf("%s/%s%s", filePath, uuid.String(), suffix)

    err = saveUploadFile(file, filename)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ResponseJson{Code: 500, Message: "文件保存失败"})
    }
    url := fmt.Sprintf("%sdownload/%s%s", host, uuid.String(), suffix)
    return c.JSON(http.StatusOK, ResponseJson{Code: 200, Url: url, Type: imgType, Message: "成功"})
}

func getNotFountByte() []byte {
    str := `<svg width="400" height="260" xmlns="http://www.w3.org/2000/svg">
  <g>
    <title>Image Not Found</title>
    <text transform="matrix(1 0 0 1 0 0)" font-weight="bold" xml:space="preserve" text-anchor="start" font-family="Noto Sans JP" font-size="40" id="svg_3" y="147.66667" x="32.73437" stroke-width="0" stroke="#000" fill="#000000">Image Not Found</text>
  </g>
</svg>`
    return []byte(str)
}

func download(c echo.Context) error {
    filename := c.Param("filename")
    if len(filename) < 36 {
        return c.Blob(http.StatusNotFound, "image/svg+xml", getNotFountByte())
    }
    id, err := uuid.Parse(filename[:36])
    if err != nil {
        return c.Blob(http.StatusNotFound, "image/svg+xml", getNotFountByte())
    }
    filePath := time.Unix(id.Time().UnixTime()).Format("200601")
    realPath := fmt.Sprintf("%s%s/%s", imagePath, filePath, filename)
    _, err = os.Stat(realPath)
    if err != nil {
        return c.Blob(http.StatusNotFound, "image/svg+xml", getNotFountByte())
    }
    return c.Inline(realPath, filename)
}

func main() {
    if len(os.Args) < 3 {
        fmt.Println("启动图床服务失败。\n请传入Host地址和图片保存路径。\n例如：\n./image-bed http://localhost:1323 /home/qiran/image-bed-path")
        return
    }
    host = os.Args[1]
    if len(host) > 0 && host[len(host)-1] != '/' {
        host += "/"
    }
    imagePath = os.Args[2]
    if len(imagePath) > 0 && imagePath[len(imagePath)-1] != '/' {
        imagePath += "/"
    }
    e := echo.New()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    e.GET("/", func(c echo.Context) error {
        return c.HTML(http.StatusOK, "<h1>这是一个图床服务。</h1>")
    })
    e.POST("/upload", upload)
    e.GET("/download/:filename", download)
    e.Logger.Fatal(e.Start(":1323"))
}
