package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	getJsonURL = "https://api.zzzmh.cn/bz/getJson"

	// Target 的类型
	TargetAnime    = "anime"
	TargetPeople   = "people"
	TargetIndex    = "index"
	TargetClassify = "classify"
)

// request 的固定 Header
var defaultHeader = http.Header{
	"User-Agent":      []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:84.0) Gecko/20100101 Firefox/84.0"},
	"Accept-Language": []string{"zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
	"Sign":            []string{"error"},
	"DNT":             []string{"1"},
	"TE":              []string{"Trailers"},
	"Pragma":          []string{"no-cache"},
	"Content-Type":    []string{"application/json"},
}

// GetJsonReq 获取图片列表的请求参数
type GetJsonReq struct {
	Target   string    `json:"target"` // const
	PageNum  uint      `json:"pageNum"`
	EndNum   uint      `json:"-"`
	TempDate *tempDate `json:"-"`
}

// tempDate 请求的临时目录
type tempDate struct {
	Current int
	Total   int
	Pages   uint
	Size    int
}

// HaveNextPage 是否有下一页
func (r *GetJsonReq) HaveNextPage() bool {
	if r.TempDate != nil && r.EndNum == 0 {
		r.EndNum = r.TempDate.Pages
	}

	if r.PageNum == r.EndNum {
		return false
	}

	if r.TempDate != nil && r.PageNum < r.TempDate.Pages && r.PageNum < r.EndNum {
		r.PageNum++
		return true
	}

	return false
}

// 返回结果
// imageType 图片的类型，后缀
type imageType string

const (
	ImagePNG imageType = "p"
	ImageJPG imageType = "j"
)

// ResultJSON 图片列表 json
type ResultJSON struct {
	Msg    string `json:"msg"`
	Code   int    `json:"code"`
	Result *struct {
		Current     int         `json:"current,omitempty"`
		Total       int         `json:"total,omitempty"`
		Pages       uint        `json:"pages,omitempty"`
		Size        int         `json:"size,omitempty"`
		Records     []*imageMsg `json:"records,omitempty"`
		SearchCount bool        `json:"searchCount,omitempty"`
		Orders      []string    `json:"orders,omitempty"`
	} `json:"result,omitempty"`
}

// imageMsg img 的信息
type imageMsg struct {
	Type     imageType `json:"t,omitempty"`
	ID       string    `json:"i,omitempty"`
	X        int       `json:"x,omitempty"`
	Y        int       `json:"y,omitempty"`
	BodyByte []byte    `json:"-"`
}

// GetURLName 获取图片的下载名称
func (img *imageMsg) GetURLName() string {
	switch img.Type {
	case ImageJPG:
		return fmt.Sprintf("%s.jpg", img.ID)
	case ImagePNG:
		return fmt.Sprintf("%s.png", img.ID)
	default:
		return ""
		// return img.ID
	}
}

// GetFileName 获取图片的文件名称
func (img *imageMsg) GetFileName() string {
	switch img.Type {
	case ImageJPG:
		return fmt.Sprintf("%s_%d_%d.jpg", img.ID, img.X, img.Y)
	case ImagePNG:
		return fmt.Sprintf("%s_%d_%d.png", img.ID, img.X, img.Y)
	default:
		return ""
		// return img.ID
	}
}

// GetGrouping 获取图片的存储分组
func (img *imageMsg) GetGrouping() string {
	if len(img.ID) == 6 {
		// 0<= string <2 , [0:2)
		return img.ID[:2]
	}
	return ""
}

// GetJson 获取指定类型的图片列表
func (r *GetJsonReq) GetJson(ctx context.Context, disposeJson func(*ResultJSON) (bool, error)) error {

	param, err := json.Marshal(r)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, getJsonURL, bytes.NewReader(param))
	if err != nil {
		return err
	}

	req.Header = defaultHeader

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	jbody := new(ResultJSON)

	err = json.Unmarshal(body, jbody)
	if err != nil {
		return err
	}

	ok, err := disposeJson(jbody)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if r.TempDate == nil && jbody.Result != nil {
		r.TempDate = &tempDate{
			Current: jbody.Result.Current,
			Total:   jbody.Result.Total,
			Pages:   jbody.Result.Pages,
			Size:    jbody.Result.Size,
		}
	}

	// runtime.GC() // 导致下载信息不完整

	return nil

}
