package httpclient

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
)

type htmlSymbol uint8

const (
	symbolJSON htmlSymbol = iota
	symbolString
)

const reportDomain = "form.hhu.edu.cn"

var (
	prefixArray = [...]string{"var _selfFormWid", "fillDetail"}
	symbolArray = [...]htmlSymbol{symbolString, symbolJSON}
	//ErrCannotParseData cannot parse html data error
	ErrCannotParseData = errors.New("data: parse error")
)

// getFormSessionID 获取打卡系统的SessionID
func (c *punchClient) getFormSessionID() (err error) {
	var req *http.Request
	req, err = getWithContext(c.ctx, "http://"+reportDomain+"/pdc/form/list")
	if err != nil {
		return
	}

	var res *http.Response
	if res, err = c.httpClient.Do(req); err != nil {
		return
	}
	drainBody(res.Body)

	if c.jar.getCookieByDomain(reportDomain) == nil {
		err = CookieNotFoundErr{"JSESSIONID"}
	}
	return
}

// matchFunc return a filter by classname
func matchFunc(class []string) func(detail reportDetail) bool {
	if len(class) == 0 {
		return func(detail reportDetail) bool {
			return true
		}
	}
	set := make(map[string]struct{}, len(class))
	for _, c := range class {
		set[c] = struct{}{}
	}
	return func(detail reportDetail) bool {
		_, ok := set[detail.class()]
		return ok
	}
}

// getFormDetail 获取打卡表单详细信息
func (c *punchClient) getFormDetail(wid string, key string, class ...string) (result detailArray, err error) {
	// match := matchFunc(class)

	form := queryForm{
		Wid:      wid,
		Date:     time.Now().In(timeZone).Format("2006-01-02"),
		Key:      key,
		Page:     1,
		PageSize: 200,
	}
	resData := queryResult{
		MaxPage: 1,
	}

	for ; form.Page <= resData.MaxPage; form.Page++ {
		var data url.Values
		data, err = query.Values(form)
		if err != nil {
			return
		}
		var req *http.Request
		req, err = postFormWithContext(c.ctx, "http://"+reportDomain+"/pdc/immediate/statisticsGrid", data)
		if err != nil {
			return
		}
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01") // accept json

		var res *http.Response
		if res, err = c.httpClient.Do(req); err != nil {
			return
		}

		decoder := json.NewDecoder(res.Body)

		err = decoder.Decode(&resData)
		drainBody(res.Body)
		if err != nil {
			return
		}

		resData.Detail.clear()

		result = append(result, resData.Detail...)
	}
	return
}
