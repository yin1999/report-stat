package httpclient

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// timeZone is used for set DataTime in HealthForm
var timeZone = time.FixedZone("CST", 8*3600)

// LoginConfirm 验证账号密码
func LoginConfirm(ctx context.Context, account *Account, timeout time.Duration) error {
	var cc context.CancelFunc
	ctx, cc = context.WithTimeout(ctx, timeout)
	c := newClient(ctx)
	err := c.login(account)
	cc()
	return parseURLError(err)
}

// GetFormData get form data
func GetFormData(ctx context.Context, account *Account) (empty bool, err error) {
	defer func() {
		err = parseURLError(err)
	}()

	c := newClient(ctx)
	err = c.login(account) // 登录，获取cookie
	if err != nil {
		return
	}
	defer c.logout()

	err = c.getFormSessionID() // 获取打卡系统的cookie
	if err != nil {
		return
	}

	var result detailArray
	result, err = c.getFormDetail(account.Wid, account.Key) // 获取打卡列表信息
	if err != nil {
		return
	}
	sort.Sort(result) // sort result
	err = storeJson(dumps{
		FormData:     result,
		ClassName:    result.classNames(),
		LastModified: time.Now().Unix(),
	}, account.File)
	if err != nil {
		return
	}
	err = generateImage(ctx, result, account, time.Now().Unix())
	empty = len(result) == 0
	return
}

// SetSslVerify when set false, insecure connection will be allowed
func SetSslVerify(verify bool) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: verify}
}

func newClient(ctx context.Context) *punchClient {
	jar := newCookieJar()
	return &punchClient{
		ctx:        ctx,
		jar:        jar,
		httpClient: &http.Client{Jar: jar},
	}
}

// parseURLError 解析URL错误
func parseURLError(err error) error {
	if v, ok := err.(*url.Error); ok {
		err = v.Err
	}
	return err
}
