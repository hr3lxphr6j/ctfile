package ctfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	urlpkg "net/url"
	"path"
	"regexp"
	"strings"

	"github.com/dimchansky/utfbom"
	"github.com/tidwall/gjson"

	"github.com/hr3lxphr6j/ctfile/utils"
)

type Type uint8

const (
	TypeFile Type = iota
	TypeFolder
)

var (
	ErrWalkAbort = errors.New("walk abort")
)

type File struct {
	Type Type
	ID   string
	Name string
	Size string
	Date string
}

type Share struct {
	UserID     int    `json:"userid"`
	FolderID   int    `json:"folder_id"`
	FileChk    string `json:"file_chk"`
	FolderName string `json:"folder_name"`
	FolderTime string `json:"folder_time"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Url        string `json:"url"`
	PageTitle  string `json:"page_title"`
}

const (
	apiEndpoint = "https://webapi.400gb.com"
	origin      = "https://545c.com"
)

type Client struct {
	hc      *http.Client
	isLogin bool
}

func NewClient() *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &Client{
		hc: &http.Client{
			Jar: jar,
		},
	}
}

func (c *Client) do(method, url string, params map[string]string, header map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	if params != nil {
		values := urlpkg.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		req.URL.RawQuery = values.Encode()
	}
	return c.hc.Do(req)
}

func (c *Client) Login(username, password string) error {
	// TODO:
	return nil
}

func (c *Client) Logout() error {
	jar, _ := cookiejar.New(nil)
	c.hc.Jar = jar
	return nil
}

func (c *Client) SetCookies(pubCookie string) error {
	u, err := urlpkg.Parse(apiEndpoint)
	if err != nil {
		return err
	}
	// TODO: verify cookie
	c.hc.Jar.SetCookies(u, []*http.Cookie{{Name: "pubcookie", Value: pubCookie}})
	c.isLogin = true
	return nil
}

func (c *Client) GetShareInfo(shareID, folderID string) (*Share, error) {
	url := fmt.Sprintf("%s%s", apiEndpoint, "/getdir.php")
	queries := map[string]string{
		"folder_id": folderID,
	}
	var passcode string
	s := strings.SplitN(shareID, "@", 2)
	if len(s) == 2 {
		passcode = s[0]
		queries["passcode"] = s[0]
		queries["d"] = s[1]
		queries["path"] = "d"
	} else {
		queries["d"] = shareID
	}
	resp, err := c.do(http.MethodGet, url, queries, map[string]string{"Origin": origin}, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StatusCode: %d", resp.StatusCode)
	}
	share := new(Share)
	if err := json.NewDecoder(utfbom.SkipOnly(resp.Body)).Decode(share); err != nil {
		return nil, err
	}
	if passcode != "" {
		key := fmt.Sprintf("pass_d%d", share.FolderID)
		exist := false
		u, _ := urlpkg.Parse(apiEndpoint)
		for _, item := range c.hc.Jar.Cookies(u) {
			if item.Name == key {
				exist = true
				break
			}
		}
		if !exist {
			c.hc.Jar.SetCookies(u, append(c.hc.Jar.Cookies(u), &http.Cookie{Name: key, Value: passcode}))
		}
	}
	return share, nil
}

func (c *Client) ParseFiles(share *Share) ([]*File, error) {
	url := fmt.Sprintf("%s%s", apiEndpoint, share.Url)
	resp, err := c.do(http.MethodGet, url, nil, map[string]string{"Origin": origin}, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StatusCode: %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	files := make([]*File, 0, 4)
	data := gjson.ParseBytes(b)
	data.Get("aaData").ForEach(func(_, value gjson.Result) bool {
		item := value.Array()
		file := &File{}
		file.Name = utils.Match1(`<a.*?>(.*?)</a>`, item[1].String())
		file.Size = item[2].String()
		file.Date = item[3].String()
		if utils.GetValueFromHTML(item[0].String(), "name") == "folder_ids[]" {
			file.Type = TypeFolder
		}
		switch file.Type {
		case TypeFile:
			file.ID = strings.Replace(utils.GetValueFromHTML(item[1].String(), "href"), "/file/", "", 1)
		case TypeFolder:
			file.ID = utils.GetValueFromHTML(item[0].String(), "value")
		}
		files = append(files, file)
		return true
	})
	return files, nil
}

func (c *Client) GetDownloadUrl(file *File) (map[string]string, error) {
	if file.Type != TypeFile {
		return nil, errors.New("this is not a file")
	}
	if !c.isLogin {
		return nil, errors.New("not login")
	}
	url := fmt.Sprintf("%s%s", apiEndpoint, "/getfile.php")
	resp, err := c.do(http.MethodGet, url,
		map[string]string{"f": file.ID},
		map[string]string{"Origin": origin}, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StatusCode: %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	result := gjson.ParseBytes(b)
	code := result.Get("code").Int()
	if code != 200 {
		return nil, errors.New(result.Get("message").String())
	}
	reg := regexp.MustCompile(`vip_(\D*)_url`)
	res := make(map[string]string)
	result.ForEach(func(key, value gjson.Result) bool {
		match := reg.FindStringSubmatch(key.String())
		if match == nil || len(match) < 2 {
			return true
		}
		res[match[1]] = value.String()
		return true
	})
	return res, nil
}

func (c *Client) walk(shareID, folderID, curPath string, handler func(curPath string, share *Share, file *File) bool) error {
	share, err := c.GetShareInfo(shareID, folderID)
	if err != nil {
		return err
	}
	files, err := c.ParseFiles(share)
	if err != nil {
		return err
	}
	for _, file := range files {
		switch file.Type {
		case TypeFolder:
			if err := c.walk(shareID, file.ID, path.Join(curPath, share.FolderName), handler); err != nil {
				return err
			}
		case TypeFile:
			if !handler(path.Join(curPath, share.FolderName), share, file) {
				return ErrWalkAbort
			}
		}
	}
	return nil
}

func (c *Client) Walk(shareID, folderID string, handler func(curPath string, share *Share, file *File) bool) error {
	return c.walk(shareID, folderID, "", handler)
}
