package teegon

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	userAgent string = "Teegon/Go"
)

var requestId = 0

type Client struct {
	Client        http.Client
	Key           string
	Server        string
	OAuthToken    string
	AlwaysUseSign bool
	serverUrl     *url.URL
	secret        string
	Timeout       int // timeout for Client second
}

type Response struct {
	Raw []byte
}

/**
 *实例化客户端对象,初始化属性
 */
func NewClient(server, key, secret string) (c *Client, err error) {
	c = &Client{
		Key:    key,
		Server: server,
		secret: secret,
	}

	c.serverUrl, err = url.ParseRequestURI(server)
	return
}

/**
 *获取全局变量requestId
 */
func GetRequestId() string {
	requestId = requestId + 1
	return fmt.Sprintf("%d", requestId)
}

func (r *Response) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Raw, v)
}

/**
 * GET请求
 * @param string api 请求方法
 * @param *map[string]interface{} 请求参数:hash键值对
 *
 * @return *Response rsp Response类型的指针
 * @return error err
 */
func (c *Client) Get(api string, params *map[string]interface{}) (rsp *Response, err error) {
	return c.do("GET", api, params)
}

/**
 * POST请求
 * @param string api 请求方法
 * @param *map[string]interface{} 请求参数:hash键值对
 *
 * @return *Response rsp Response类型的指针
 * @return error err
 */
func (c *Client) Post(api string, params *map[string]interface{}) (rsp *Response, err error) {
	return c.do("POST", api, params)
}

/**
 * PUT请求
 * @param string api 请求方法
 * @param *map[string]interface{} 请求参数:hash键值对
 *
 * @return *Response rsp Response类型的指针
 * @return error err
 */
func (c *Client) Put(api string, params *map[string]interface{}) (rsp *Response, err error) {
	return c.do("PUT", api, params)
}

/**
 * DELETE请求
 * @param string api 请求方法
 * @param *map[string]interface{} 请求参数:hash键值对
 *
 * @return *Response rsp Response类型的指针
 * @return error err
 */
func (c *Client) Delete(api string, params *map[string]interface{}) (rsp *Response, err error) {
	return c.do("DELETE", api, params)
}

/**
 * do发起请求
 * @param string method 请求方式 GET|POST|PUT|DELETE
 * @param string api 请求方法
 * @param *map[string]interface{} 请求参数:hash键值对
 *
 * @return *Response rsp Response类型的指针
 * @return error err
 */
func (c *Client) do(method, api string, params *map[string]interface{}) (rsp *Response, err error) {
	r, err := c.getRequest(method, api, params)
	if err != nil {
		return nil, err
	}
	res, err := c.Client.Do(r)

	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	return &Response{data}, err
}


/**
 * getRequest 生成请求对象
 * @param string method 请求方式 GET|POST|PUT|DELETE
 * @param string api 请求方法
 * @param *map[string]interface{} 请求参数:hash键值对
 *
 * @return *http.Request 类型指针
 * @return error err
 */
func (c *Client) getRequest(method, api string, params *map[string]interface{}) (req *http.Request, err error) {
	vals := url.Values{}

	if params != nil {
		for k, v := range *params {
			vals.Set(k, paramToStr(v))
		}
	}

	r, err := http.NewRequest(method, c.Server, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Set("User-Agent", userAgent)
	if c.OAuthToken != "" {
		r.Header.Set("Authorization", "Bearer "+c.OAuthToken)
	}

	use_url_query := method != "POST"
	vals.Set("method", api)
	vals.Set("app_key", c.Key)

	if !c.AlwaysUseSign && r.URL.Scheme == "https" {
		tr := &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: true,
		}
		c.Client.Transport = tr

		vals.Set("client_secret", c.secret)

	} else {
		vals.Set("sign_time", strconv.FormatInt(time.Now().Unix(), 10))
		if use_url_query {
			r.URL.RawQuery = vals.Encode()
		} else {
			r.PostForm = vals
		}
		vals.Set("sign", Sign(r, c.secret))
	}

	// we just need timeout forever
	c.Client.Timeout = time.Duration(int64(c.Timeout)) * time.Second

	query_string := vals.Encode()

	if use_url_query {
		r.URL.RawQuery = query_string
	} else {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.ContentLength = int64(len(query_string))
		r.Body = &closebuf{bytes.NewBufferString(query_string)}
	}

	return r, nil
}

func paramToStr(v interface{}) (v2 string) {
	switch v.(type) {
	case string:
		v2 = v.(string)
	default:
		buf, _ := json.Marshal(v)
		v2 = string(buf)
	}
	return
}

type closebuf struct {
	*bytes.Buffer
}

func (cb *closebuf) Close() error {
	return nil
}
