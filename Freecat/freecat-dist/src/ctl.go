package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ctlClient struct {
	base   string
	secret string
	hc     *http.Client
}

func newCtl(port, secret string) *ctlClient {
	ctl := port
	if ctl == "" {
		ctl = "9090"
	}
	return &ctlClient{
		base:   "http://127.0.0.1:" + strings.TrimPrefix(ctl, ":"),
		secret: secret,
		hc:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ctlClient) do(method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
	if err != nil {
		return err
	}
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("连接核心失败（是否已运行 `freecat run`？）: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("核心返回 HTTP %d: %s", resp.StatusCode, shortBody(data))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

type proxyInfo struct {
	Type    string   `json:"type"`
	Now     string   `json:"now"`
	All     []string `json:"all"`
	History []struct {
		Delay int `json:"delay"`
	} `json:"history"`
}

func (c *ctlClient) proxies() (map[string]proxyInfo, error) {
	var r struct {
		Proxies map[string]proxyInfo `json:"proxies"`
	}
	err := c.do(http.MethodGet, "/proxies", nil, &r)
	return r.Proxies, err
}

func (c *ctlClient) selectNode(group, node string) error {
	return c.do(http.MethodPut, "/proxies/"+url.PathEscape(group), map[string]string{"name": node}, nil)
}

func (c *ctlClient) delay(node string) (int, error) {
	var r struct {
		Delay int `json:"delay"`
	}
	u := "/proxies/" + url.PathEscape(node) + "/delay?timeout=5000&url=" + url.QueryEscape("http://cp.cloudflare.com/generate_204")
	if err := c.do(http.MethodGet, u, nil, &r); err != nil {
		return 0, err
	}
	return r.Delay, nil
}

func (c *ctlClient) version() (string, error) {
	var r struct {
		Version string `json:"version"`
	}
	err := c.do(http.MethodGet, "/version", nil, &r)
	return r.Version, err
}

func (c *ctlClient) configs() (map[string]any, error) {
	var r map[string]any
	err := c.do(http.MethodGet, "/configs", nil, &r)
	return r, err
}
