package request

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/bitly/go-simplejson"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Request struct {
	Type        string
	Url         string
	Head        http.Header
	ConnTimeout time.Duration
	Timeout     time.Duration
	Times       int
	Delay       time.Duration
	Char        string
	Data        url.Values
}

func NewDocument(request *Request) (*goquery.Document, error) {
	r := *request
	res, err := TryNewRequest(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var rd io.Reader
	if r.Char != "" || r.Char == "utf-8" {
		dec := mahonia.NewDecoder(r.Char)
		rd = dec.NewReader(res.Body)
	} else {
		rd = res.Body
	}

	return goquery.NewDocumentFromReader(rd)
}

// 打开网页
func NewRequest(request *Request) (*http.Response, error) {
	r := *request

	connTimeout := r.ConnTimeout // 连接超时时间
	timeout := r.Timeout         // 传输请求时间

	// 默认时间
	if connTimeout == 0 {
		connTimeout = 5
	}
	if timeout == 0 {
		timeout = 10
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Second*connTimeout) // 设置建立连接超时
				if err != nil {
					return nil, err
				}
				c.SetDeadline(time.Now().Add(timeout * time.Second)) // 设置发送接收数据超时
				return c, nil
			},
		},
	}

	// form
	formio := r.Data.Encode()
	form := strings.NewReader(formio)

	// 请求类型
	typeIs := r.Type
	if typeIs == "" {
		typeIs = "GET"
	}

	req, err := http.NewRequest(typeIs, r.Url, form) // 发起请求

	// header
	req.Header = r.Head
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// 尝试打开网页
func TryNewRequest(request *Request) (*http.Response, error) {
	r := *request

	// 尝试次数
	times := r.Times
	if times == 0 {
		times = 3
	}

	// 重试间隔时间
	delay := r.Delay
	if delay == 0 {
		delay = 3
	}

	for i := 1; i <= times; i++ {
		res, err := NewRequest(request)
		if err == nil {
			return res, err
		} else {
			fmt.Printf("请求失败，正在重试：%d\r\n", i)
		}
		time.Sleep(delay * time.Second) // 重试间隔
	}
	err := errors.New("打开网页失败")
	return nil, err
}

// 请求json
func NewJson(request *Request) (*simplejson.Json, error) {
	r := *request

	// 尝试次数
	times := r.Times
	if times == 0 {
		times = 3
	}

	var res *http.Response
	var json *simplejson.Json
	var err error
	for i := 1; i <= times; i++ {
		res, err = TryNewRequest(request)
		if err != nil {
			return nil, err
		}

		body, _ := ioutil.ReadAll(res.Body)

		json, err = simplejson.NewJson(body)
		if err == nil {
			return json, err
		} else {
			fmt.Printf("无效数据，重新获取：%d\r\n", i)
		}
	}

	res.Body.Close()
	return nil, err
}
