package sub

import (
	"tbox/log"
	"crypto/md5"
	"fmt"
	"net/http"
)

type Subscirbe struct {
	Name      string `json:"name"`
	Url       string `json:"url"`
	Using     bool   `json:"using"`
	UserAgent string `json:"user_agent"`
	Remark    string `json:"remark"`
}

// NewSubscirbe creates a new Subscribe with the given url and name.
func NewSubscirbe(url, name string) *Subscirbe {
	return &Subscirbe{Name: name, Url: url, Using: true}
}

func (s *Subscirbe) ID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s.Url)))
}

func (s *Subscirbe) UpdataNode(opt UpdataOption) []string {
	var res *http.Response
	var err error
	ua := opt.userAgent()
	if s.UserAgent != "" {
		ua = s.UserAgent
	}
	switch opt.proxyMode() {
	case SOCKS:
		res, err = GetBySocks5Proxy(s.Url, opt.addr(), opt.port(), opt.timeout(), ua)
	case HTTP:
		res, err = GetByHTTPProxy(s.Url, opt.addr(), opt.port(), opt.timeout(), ua)
	default:
		res, err = GetNoProxy(s.Url, opt.timeout(), ua)
	}
	if err != nil {
		log.Error(err)
		return []string{}
	}
	log.Info("fetched [", s.Url, "] -- ", res.Status)
	text := ReadDate(res)
	res.Body.Close()
	return Sub2links(text)
}
