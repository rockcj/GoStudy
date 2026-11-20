package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// hosts are the client CDN endpoints published by the Freecat app.
// All of them serve the same v2board-compatible API.
var defaultHosts = []string{
	"https://tccdnclient.miaowuppaa.com",
	"https://tuc.skyoutdoors.com.cn",
	"https://115.159.221.23",
	"https://us.freecat.cloud",
}

const clashUA = "ClashforWindows/0.20.39"

func newClient(timeout time.Duration) *http.Client {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // some hosts are bare-IP endpoints
		Proxy:               http.ProxyFromEnvironment,
		ForceAttemptHTTP2:   true,
		MaxIdleConnsPerHost: 4,
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

type loginResp struct {
	Data struct {
		Token    string `json:"token"`
		AuthData string `json:"auth_data"`
		IsAdmin  int    `json:"is_admin"`
	} `json:"data"`
	Message string `json:"message"`
}

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }

// login tries every known host until one returns a token.
func login(email, password, preferHost string) (host, token, authData string, err error) {
	hosts := defaultHosts
	if preferHost != "" {
		hosts = append([]string{strings.TrimRight(preferHost, "/")}, defaultHosts...)
	}
	var lastErr error
	seen := map[string]bool{}
	for _, h := range hosts {
		h = strings.TrimRight(h, "/")
		if seen[h] {
			continue
		}
		seen[h] = true
		tok, auth, e := loginAt(h, email, password)
		if e == nil {
			return h, tok, auth, nil
		}
		lastErr = fmt.Errorf("%s: %w", h, e)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("没有可用的服务器")
	}
	return "", "", "", lastErr
}

func loginAt(host, email, password string) (string, string, error) {
	form := url.Values{}
	form.Set("email", email)
	form.Set("password", password)
	req, err := http.NewRequest(http.MethodPost, host+"/api/v1/passport/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", clashUA)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", host)
	req.Header.Set("Referer", host+"/")

	resp, err := newClient(20 * time.Second).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", "", &apiError{fmt.Sprintf("HTTP %d: %s", resp.StatusCode, shortBody(body))}
	}
	var lr loginResp
	if err := json.Unmarshal(body, &lr); err != nil {
		return "", "", &apiError{"返回不是 JSON: " + shortBody(body)}
	}
	if lr.Data.Token == "" {
		msg := lr.Message
		if msg == "" {
			msg = shortBody(body)
		}
		return "", "", &apiError{"登录失败: " + msg}
	}
	return lr.Data.Token, lr.Data.AuthData, nil
}

// fetchSubscription downloads the Clash-format profile.
func fetchSubscription(host, token string) (body []byte, info string, err error) {
	u := strings.TrimRight(host, "/") + "/api/v1/client/subscribe?token=" + url.QueryEscape(token)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", clashUA)
	req.Header.Set("Accept", "*/*")
	resp, err := newClient(60 * time.Second).Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", &apiError{fmt.Sprintf("HTTP %d: %s", resp.StatusCode, shortBody(b))}
	}
	if !strings.Contains(string(b), "proxies:") {
		return nil, "", &apiError{"订阅内容不是 Clash 配置: " + shortBody(b)}
	}
	return b, resp.Header.Get("subscription-userinfo"), nil
}

func shortBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 180 {
		s = s[:180] + "..."
	}
	return s
}

// httpGetViaProxy fetches a URL through the local HTTP proxy.
func httpGetViaProxy(proxy, target string) ([]byte, error) {
	pu, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{Proxy: http.ProxyURL(pu)}
	c := &http.Client{Timeout: 25 * time.Second, Transport: tr}
	resp, err := c.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 400 {
		return nil, &apiError{fmt.Sprintf("HTTP %d: %s", resp.StatusCode, shortBody(b))}
	}
	return b, nil
}

// prettyTraffic turns subscription-userinfo into a short human summary.
func prettyTraffic(info string) string {
	if info == "" {
		return ""
	}
	get := func(k string) int64 {
		for _, part := range strings.Split(info, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, k+"=") {
				var n int64
				fmt.Sscanf(strings.TrimPrefix(part, k+"="), "%d", &n)
				return n
			}
		}
		return 0
	}
	up, down, total := get("upload"), get("download"), get("total")
	expire := get("expire")
	gb := func(n int64) string { return fmt.Sprintf("%.2f GB", float64(n)/1024/1024/1024) }
	s := fmt.Sprintf("已用 %s / %s（上传 %s，下载 %s）", gb(up+down), gb(total), gb(up), gb(down))
	if expire > 0 {
		s += "，到期 " + time.Unix(expire, 0).Format("2006-01-02")
	}
	return s
}
