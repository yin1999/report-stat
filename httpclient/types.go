package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// QueryParam query param struct
type queryForm struct {
	Wid        string `url:"wid"`
	Department string `url:"dept"`
	Date       string `url:"inputDate"`
	Key        string `url:"key"`
	Page       uint   `url:"page"`
	PageSize   uint   `url:"pagesize"`
}

type dumps struct {
	ClassName    []string    `json:"className"`
	FormData     detailArray `json:"formData"`
	LastModified int64       `json:"lastModified"`
}

type reportDetail [8]string // date, status, _, id, name, college, grade, class, _phone

var _ json.Marshaler = reportDetail{}

func (da detailArray) clear() {
	for i := range da {
		da[i].clear()
	}
}

// clear remove the leading character 'C' for changzhou
func (rd *reportDetail) clear() {
	if rd[7][0] == 'C' {
		rd[7] = rd[7][1:]
	}
}

func (rd reportDetail) MarshalJSON() ([]byte, error) {
	return sliceToJson(rd[3:]), nil
}

func sliceToJson(slice []string) []byte {
	n := len(slice)*3 + 1
	for i := range slice {
		n += len(slice[i])
	}
	buf := bytes.Buffer{}
	buf.Grow(n)
	buf.WriteString("[\"")
	buf.WriteString(slice[0])
	for _, s := range slice[1:] {
		buf.WriteString("\",\"")
		buf.WriteString(s)
	}
	buf.WriteString("\"]")
	return buf.Bytes()
}

func (rd reportDetail) grade() string {
	return rd[6]
}

func (rd reportDetail) id() string {
	return rd[3]
}

func (rd reportDetail) name() string {
	return rd[4]
}
func (rd reportDetail) class() string {
	return rd[7]
}

type detailArray []reportDetail

type queryResult struct {
	CurrentPage uint        `json:"curPage"`
	IsReported  bool        `json:"isReported"`
	Detail      detailArray `json:"jexcelDatas"`
	MaxPage     uint        `json:"maxPage"`
	TotalNum    uint        `json:"totalNum"`
}

func (arr detailArray) Len() int {
	return len(arr)
}

func (arr detailArray) Less(x, y int) bool {
	c1, c2 := arr[x].class(), arr[y].class()
	if c1 == c2 {
		return arr[x].id() < arr[y].id()
	}
	return c1 < c2
}

func (arr detailArray) Swap(x, y int) {
	arr[x], arr[y] = arr[y], arr[x]
}

// classNames get all class name
//
// Note: arr must be sorted
func (arr detailArray) classNames() []string {
	if len(arr) == 0 {
		return make([]string, 0)
	}
	tmp := arr[0].class()
	res := []string{tmp}
	for i := range arr {
		if arr[i].class() != tmp {
			tmp = arr[i].class()
			res = append(res, tmp)
		}
	}
	return res
}

func (arr detailArray) filter(f func(detail reportDetail) bool) detailArray {
	if len(arr) == 0 {
		return arr
	}
	var result detailArray
	var i int
	var v reportDetail
	for i, v = range arr {
		if !f(v) {
			result = append(result, arr[:i]...)
		}
	}
	if len(result) == 0 { // all matched, return a copy of original arr
		result = make(detailArray, len(arr))
		copy(result, arr)
	} else {
		arr = arr[i+1:]
		for _, v = range arr {
			if f(v) {
				result = append(result, v)
			}
		}
	}
	return result
}

// CookieNotFoundErr error interface for Cookies
type CookieNotFoundErr struct {
	cookie string
}

func (t CookieNotFoundErr) Error() string {
	return "http: can't find cookie: " + t.cookie
}

type punchClient struct {
	ctx        context.Context
	httpClient *http.Client
	jar        *cookieJar
}

// Account account info for login
type Account struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Class    []string `json:"class"`
	Wid      string   `json:"wid"`
	Key      string   `json:"key"`
	File     string   `json:"file"`
	Out      string   `json:"out"`
}

// Name get the name of the account
func (a Account) Name() string {
	return a.Username
}
