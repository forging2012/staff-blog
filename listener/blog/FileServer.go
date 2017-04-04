package blog

import (
	"fmt"
	"github.com/num5/axiom"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"
	"encoding/json"
	"path/filepath"
	"strings"
)

type FileHandler struct {
	tplPath  string
	savePath string
	ctx      *axiom.Context
}

func newFileHandler(tpl, save string, ctx *axiom.Context) *FileHandler {
	return &FileHandler{
		tplPath:  tpl,
		savePath: save,
		ctx:      ctx,
	}
}

func (fh *FileHandler) Http() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(fh.tplPath+"/assets/"))))
	http.Handle("/file", http.StripPrefix("/file/", http.FileServer(http.Dir(fh.savePath))))
	http.HandleFunc("/", fh.index)
	http.HandleFunc("/upload", fh.upload)
	http.HandleFunc("/files", fh.filewolk)
	fh.ctx.Reply("文件上传服务监听端口：%d", 8800)
	err := http.ListenAndServe(":8800", nil)
	if err != nil {
		fh.ctx.Reply("开启文件上传服务器错误：%s", err.Error())
	}
}

func (fh *FileHandler) index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(fh.tplPath + "/index.html")
	if err != nil {
		fh.ctx.Reply("解析主页模版失败：%s", err)
	}
	err = t.Execute(w, "上传文件")
	if err != nil {
		fh.ctx.Reply("解析主页模版失败：%s", err)
	}
}

// 上传文件接口
func (fh *FileHandler) upload(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("文件上传异常:%s\n", err)
		}
	}()

	if "POST" == r.Method {

		r.ParseMultipartForm(32 << 20) //在使用r.MultipartForm前必须先调用ParseMultipartForm方法，参数为最大缓存

		file, handler, err := r.FormFile("file")
		if err != nil {
			fh.ctx.Reply("未找到上传文件：%s", err)
			resp := map[string]interface{} {
				"code": 500,
				"error": "未找到上传文件:"+err.Error(),
			}
			out, _ := json.Marshal(resp)
			w.Write(out)
			return
		}

		filename := handler.Filename

		save := fh.savePath + "/" + filename

		//检查文件是否存在
		if !Exist(fh.savePath) {
			os.MkdirAll(fh.savePath, os.ModePerm)
		} else {
			if Exist(save) {
				fh.ctx.Reply("博客《%s》文件已经存在", filename)
				resp := map[string]interface{} {
					"code": 500,
					"error": "博客《"+filename+"》文件已经存在",
				}
				out, _ := json.Marshal(resp)
				w.Write(out)
				return
			}
		}

		//结束文件
		of, err := handler.Open()
		if err != nil {
			fh.ctx.Reply("文件处理错误： %s", err)
			resp := map[string]interface{} {
				"code": 500,
				"error": "文件处理错误:"+err.Error(),
			}
			out, _ := json.Marshal(resp)
			w.Write(out)
			return
		}
		defer file.Close()

		//保存文件
		f, err := os.Create(save)
		if err != nil {
			fh.ctx.Reply("创建文件失败： %s", err)
			resp := map[string]interface{} {
				"code": 500,
				"error": "创建文件失败:"+err.Error(),
			}
			out, _ := json.Marshal(resp)
			w.Write(out)
			return
		}
		defer f.Close()
		io.Copy(f, of)

		//获取文件状态信息
		fstat, _ := f.Stat()

		//打印接收信息
		print := fmt.Sprintf("上传时间:%s, Size: %dKB,  Name:%s\n", time.Now().Format("2006-01-02 15:04:05"), fstat.Size()/1024, filename)
		fh.ctx.Reply(print)

		resp := map[string]interface{} {
			"code": 0,
			"msg": print,
		}
		out, _ := json.Marshal(resp)
		w.Write(out)

		return
	}
}

func (fh *FileHandler) filewolk(w http.ResponseWriter, r *http.Request) {
	dir := fh.savePath
	var filemaps []string
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if ( f == nil ) {
			return err
		}
		if f.IsDir() {
			return nil
		}

		filename := strings.TrimLeft(path, fh.savePath)

		filemaps = append(filemaps, filename)

		return nil
	})
	if err != nil {
		w.Write([]byte("filepath.Walk() returned" + err.Error()))
	}

	out, err := json.Marshal(filemaps)
	w.Write(out)
}

func (fh *FileHandler) delete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filename := r.FormValue("filename")

	file := fh.savePath + "/" + filename

	err := os.Remove(file)
	if err != nil {
		resp := map[string]interface{} {
			"code": 500,
			"error": "删除文件失败:"+err.Error(),
		}
		out, _ := json.Marshal(resp)
		w.Write(out)
		return
	}

	resp := map[string]interface{} {
		"code": 0,
		"error": "删除《" + filename + "》文件成功",
	}
	out, _ := json.Marshal(resp)
	w.Write(out)
	return
}

