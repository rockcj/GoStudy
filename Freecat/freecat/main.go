package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const version = "0.2.0"

// defaultServerHost is the verified Freecat entry host. It is a DNS name (not
// a fixed IP) so the vendor can rotate the real servers behind it.
const defaultServerHost = "aqaenca1.tobigyang.cn"

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "login":
		cmdLogin(args)
	case "update":
		cmdUpdate(args)
	case "run":
		cmdRun(args)
	case "nodes":
		cmdNodes(args)
	case "use":
		cmdUse(args)
	case "status":
		cmdStatus(args)
	case "test":
		cmdTest(args)
	case "env":
		cmdEnv(args)
	case "dns":
		cmdDNS(args)
	case "version", "-v", "--version":
		fmt.Println("freecat", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`freecat - 自由猫 Linux 命令行客户端

用法:
  freecat login  -email 账号 -password 密码     登录并保存令牌
  freecat update                                拉取/更新订阅与节点
  freecat run    [--tun] [--node 节点]          启动内核（默认仅本地代理）
  freecat nodes  [--group 组名] [--all]         列出策略组与节点
  freecat use    序号|节点名|auto                手动切换节点
  freecat status                                查看状态/实际出口节点/流量
  freecat test                                  通过本地代理做连通性测试
  freecat env                                   打印可 source 的代理环境变量
  freecat dns on|off|status                     配合 TUN：把系统 DNS 指向本机内核

常用参数:
  --tun             启用 TUN 全局透明代理（需要 root）
  --node 名称/序号    启动时自动切到指定节点（"auto" 恢复自动选择）
  --port N          HTTP/SOCKS 混合端口，默认 7890
  --ctl PORT        内核控制端口，默认 9090
  --core PATH       指定 mihomo 内核路径
  --server-host H   节点入口（域名或 IP，默认 aqaenca1.tobigyang.cn；填 keep 保留订阅原域名）

配置文件目录: $FREECAT_HOME 或 ~/.config/freecat
`)
}

func cmdLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	email := fs.String("email", os.Getenv("FREECAT_EMAIL"), "账号邮箱")
	pass := fs.String("password", os.Getenv("FREECAT_PASSWORD"), "密码")
	host := fs.String("host", "", "指定服务器（默认自动选择）")
	_ = fs.Parse(args)

	e, p := *email, *pass
	rd := bufio.NewReader(os.Stdin)
	if e == "" {
		fmt.Print("邮箱: ")
		e = readLine(rd)
	}
	if p == "" {
		fmt.Print("密码: ")
		p = readLine(rd)
	}
	if e == "" || p == "" {
		fatal("邮箱和密码不能为空")
	}

	fmt.Println("正在登录…")
	h, token, authData, err := login(e, p, *host)
	if err != nil {
		fatal("登录失败: %v", err)
	}
	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	st.Email, st.Token, st.AuthData, st.Host = e, token, authData, h
	if st.Port == 0 {
		st.Port = 7890
	}
	if st.Controller == "" {
		st.Controller = "9090"
	}
	if st.Secret == "" {
		st.Secret = randomSecret()
	}
	if err := saveState(st); err != nil {
		fatal("保存状态失败: %v", err)
	}
	fmt.Printf("登录成功: %s\n服务器: %s\n", maskEmail(e), h)
	fmt.Println("下一步: freecat update")
}

func cmdUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	host := fs.String("host", "", "指定服务器")
	serverHost := fs.String("server-host", "", "节点入口（域名或 IP）")
	serverIP := fs.String("server-ip", "", "同 --server-host（兼容旧参数）")
	_ = fs.Parse(args)

	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	if st.Token == "" {
		fatal("尚未登录，请先运行: freecat login -email 你的邮箱 -password 你的密码")
	}
	switch {
	case *serverHost != "":
		st.ServerHost, st.ServerIP = *serverHost, ""
	case *serverIP != "":
		st.ServerHost, st.ServerIP = *serverIP, ""
	case st.ServerHost == "" && st.ServerIP == "":
		st.ServerHost = defaultServerHost
	}

	useHost := st.Host
	if *host != "" {
		useHost = *host
	}
	if useHost == "" {
		useHost = defaultHosts[0]
	}
	fmt.Printf("从 %s 拉取订阅…\n", useHost)
	body, info, err := fetchSubscription(useHost, st.Token)
	if err != nil {
		var lastErr error = err
		for _, h := range defaultHosts {
			if h == useHost {
				continue
			}
			fmt.Printf("  重试 %s …\n", h)
			b, i, e2 := fetchSubscription(h, st.Token)
			if e2 == nil {
				body, info, useHost, err = b, i, h, nil
				break
			}
			lastErr = e2
		}
		if err != nil {
			fatal("拉取订阅失败: %v", lastErr)
		}
	}
	if err := writeFile0600(profilePath(), body); err != nil {
		fatal("保存订阅失败: %v", err)
	}
	st.Host = useHost
	st.UpdatedAt = time.Now().Unix()
	if t := prettyTraffic(info); t != "" {
		st.Traffic = t
	}
	if st.Port == 0 {
		st.Port = 7890
	}
	if st.Controller == "" {
		st.Controller = "9090"
	}
	if st.Secret == "" {
		st.Secret = randomSecret()
	}
	if err := writeRuntime(st); err != nil {
		fatal("生成运行配置失败: %v", err)
	}
	if err := saveState(st); err != nil {
		fatal("保存状态失败: %v", err)
	}
	fmt.Printf("订阅已更新（%d 字节）\n", len(body))
	if st.Traffic != "" {
		fmt.Println("流量:", st.Traffic)
	}
	if ep := st.endpoint(); ep != "" {
		fmt.Printf("节点入口: %s\n", ep)
	} else {
		fmt.Println("节点入口: 保留订阅原域名")
	}
	fmt.Println("下一步: freecat run   （全局透明代理加 --tun，需要 root）")
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	tun := fs.Bool("tun", false, "启用 TUN")
	port := fs.Int("port", 0, "混合代理端口")
	ctl := fs.String("ctl", "", "控制端口")
	core := fs.String("core", "", "内核路径")
	serverHost := fs.String("server-host", "", "节点入口")
	serverIP := fs.String("server-ip", "", "同 --server-host")
	node := fs.String("node", "", "启动后切换到的节点（名称/序号/auto）")
	noUpdate := fs.Bool("no-update", false, "不自动更新订阅")
	_ = fs.Parse(args)

	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	if *port != 0 {
		st.Port = *port
	}
	if st.Port == 0 {
		st.Port = 7890
	}
	if *ctl != "" {
		st.Controller = *ctl
	}
	if st.Controller == "" {
		st.Controller = "9090"
	}
	if st.Secret == "" {
		st.Secret = randomSecret()
	}
	switch {
	case *serverHost != "":
		st.ServerHost, st.ServerIP = *serverHost, ""
	case *serverIP != "":
		st.ServerHost, st.ServerIP = *serverIP, ""
	}
	st.TUN = *tun

	if _, err := os.Stat(profilePath()); err != nil {
		if *noUpdate {
			fatal("没有订阅文件，请先运行 freecat update")
		}
		if st.Token == "" {
			fatal("没有订阅文件，也没有登录令牌。请先 freecat login / freecat update")
		}
		fmt.Println("未找到订阅，先自动拉取…")
		host := st.Host
		if host == "" {
			host = defaultHosts[0]
		}
		body, info, e := fetchSubscription(host, st.Token)
		if e != nil {
			fatal("拉取订阅失败: %v", e)
		}
		if e := writeFile0600(profilePath(), body); e != nil {
			fatal("%v", e)
		}
		if t := prettyTraffic(info); t != "" {
			st.Traffic = t
		}
	}
	if err := writeRuntime(st); err != nil {
		fatal("生成运行配置失败: %v", err)
	}
	if err := saveState(st); err != nil {
		fmt.Fprintln(os.Stderr, "提示: 无法写入状态文件（sudo 运行时属正常，可忽略）")
	}
	chownToSudo(runtimePath(), profilePath(), statePath(), configDir())

	corePath, err := findCore(*core)
	if err != nil {
		fatal("%v", err)
	}
	ensureGeo(corePath)
	if *tun && !isRoot() {
		fmt.Fprintln(os.Stderr, "提示: TUN 模式通常需要 root（sudo freecat run --tun）")
	}

	fmt.Printf("内核: %s\n", corePath)
	fmt.Printf("代理: http://127.0.0.1:%d  (HTTP/SOCKS 混合)\n", st.Port)
	fmt.Printf("控制: http://127.0.0.1:%s\n", st.Controller)
	if ep := st.endpoint(); ep != "" {
		fmt.Printf("节点入口: %s\n", ep)
	}
	if *node != "" {
		fmt.Printf("启动节点: %s\n", *node)
	}
	if *tun {
		fmt.Println("模式: TUN 全局透明代理")
	} else {
		fmt.Printf("临时使用: export https_proxy=http://127.0.0.1:%d http_proxy=http://127.0.0.1:%d\n", st.Port, st.Port)
	}
	fmt.Println("按 Ctrl+C 停止")

	if *node != "" {
		go func() {
			selected, err := selectNode(st, *node)
			if err != nil {
				fmt.Fprintf(os.Stderr, "节点切换失败: %v\n", err)
				return
			}
			fmt.Printf("已切换节点: %s\n", selected)
		}()
	}

	c := exec.Command(corePath, "-d", coreDir(), "-f", runtimePath())
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := c.Run(); err != nil {
		fatal("内核退出: %v", err)
	}
}

func cmdNodes(args []string) {
	fs := flag.NewFlagSet("nodes", flag.ExitOnError)
	group := fs.String("group", "", "只看指定策略组")
	all := fs.Bool("all", false, "展开所有组的全部节点")
	_ = fs.Parse(args)

	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	proxies, err := newCtl(st.Controller, st.Secret).proxies()
	if err != nil {
		fatal("%v", err)
	}
	if *group != "" {
		p, ok := proxies[*group]
		if !ok {
			fatal("找不到策略组 %q", *group)
		}
		printGroup(*group, p, proxies, true)
		return
	}
	for _, n := range groupNames(proxies) {
		p := proxies[n]
		list := *all || p.Type == "Selector"
		printGroup(n, p, proxies, list)
	}
}

func cmdUse(args []string) {
	fs := flag.NewFlagSet("use", flag.ExitOnError)
	group := fs.String("group", "", "策略组")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		fatal("用法: freecat use 序号|节点名|auto")
	}
	want := strings.Join(fs.Args(), " ")

	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	if *group != "" {
		if err := selectInGroup(st, *group, want); err != nil {
			fatal("切换失败: %v", err)
		}
		return
	}
	selected, err := selectNode(st, want)
	if err != nil {
		fatal("切换失败: %v", err)
	}
	fmt.Printf("已切换到: %s\n", selected)
}

func cmdStatus(args []string) {
	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	ctl := newCtl(st.Controller, st.Secret)
	v, err := ctl.version()
	if err != nil {
		fmt.Println("状态: 未运行（", err, "）")
	} else {
		fmt.Printf("状态: 运行中，内核 %s\n", v)
		if cfg, err := ctl.configs(); err == nil {
			fmt.Printf("模式: %v  混合端口: %v\n", cfg["mode"], cfg["mixed-port"])
		}
		if ps, err := ctl.proxies(); err == nil {
			if g := pickGroup(ps); g != "" {
				now := ps[g].Now
				actual := resolveNode(ps, now)
				if actual != now {
					fmt.Printf("当前节点: %s  [%s → %s]\n", actual, g, now)
				} else {
					fmt.Printf("当前节点: %s  [%s]\n", actual, g)
				}
			}
		}
	}
	fmt.Printf("服务器: %s\n", st.Host)
	if ep := st.endpoint(); ep != "" {
		fmt.Printf("节点入口: %s\n", ep)
	}
	if st.Traffic != "" {
		fmt.Println("流量:", st.Traffic)
	}
	if st.UpdatedAt > 0 {
		fmt.Println("订阅更新:", time.Unix(st.UpdatedAt, 0).Format("2006-01-02 15:04:05"))
	}
}

func cmdTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	u := fs.String("url", "http://cp.cloudflare.com/generate_204", "测试地址")
	_ = fs.Parse(args)

	st, err := loadState()
	if err != nil {
		fatal("%v", err)
	}
	port := st.Port
	if port == 0 {
		port = 7890
	}
	proxy := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Printf("通过 %s 请求 %s …\n", proxy, *u)
	start := time.Now()
	body, err := httpGetViaProxy(proxy, *u)
	if err != nil {
		fatal("失败: %v", err)
	}
	fmt.Printf("成功: 收到 %d 字节，耗时 %d ms\n", len(body), time.Since(start).Milliseconds())
	if ip, err := httpGetViaProxy(proxy, "https://api.ipify.org"); err == nil {
		fmt.Printf("出口 IP: %s\n", strings.TrimSpace(string(ip)))
	}
}

const (
	resolvConf   = "/etc/resolv.conf"
	resolvBak    = "/etc/resolv.conf.freecat.bak"
	resolvedConf = "/etc/systemd/resolved.conf.d/freecat.conf"
)

func cmdDNS(args []string) {
	if len(args) < 1 {
		fatal("用法: sudo freecat dns on|off|status")
	}
	action := args[0]
	if action != "status" && !isRoot() {
		fatal("需要 root: sudo freecat dns %s", action)
	}
	switch action {
	case "on":
		dnsOn()
	case "off":
		dnsOff()
	case "status":
		dnsStatus()
	default:
		fatal("未知操作: %s（可用 on/off/status）", action)
	}
}

func systemdResolvedActive() bool {
	out, err := exec.Command("systemctl", "is-active", "systemd-resolved").Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

func dnsOn() {
	if systemdResolvedActive() {
		_ = os.MkdirAll("/etc/systemd/resolved.conf.d", 0o755)
		content := "[Resolve]\nDNS=127.0.0.1\nFallbackDNS=\n"
		if err := os.WriteFile(resolvedConf, []byte(content), 0o644); err != nil {
			fatal("写入 %s 失败: %v", resolvedConf, err)
		}
		_ = exec.Command("systemctl", "restart", "systemd-resolved").Run()
		_ = exec.Command("resolvectl", "flush-caches").Run()
		fmt.Println("已配置 systemd-resolved 使用 127.0.0.1（freecat 内核 DNS）")
		fmt.Println("关闭: sudo freecat dns off")
		return
	}
	b, err := os.ReadFile(resolvConf)
	if err != nil {
		b = []byte("nameserver 223.5.5.5\n")
	}
	if _, err := os.Stat(resolvBak); err != nil {
		_ = os.WriteFile(resolvBak, b, 0o644)
	}
	_ = os.Remove(resolvConf)
	if err := os.WriteFile(resolvConf, []byte("nameserver 127.0.0.1\n"), 0o644); err != nil {
		fatal("写入 %s 失败: %v", resolvConf, err)
	}
	fmt.Println("已把 /etc/resolv.conf 指向 127.0.0.1（原文件备份为", resolvBak, "）")
	fmt.Println("关闭: sudo freecat dns off")
}

func dnsOff() {
	if _, err := os.Stat(resolvedConf); err == nil {
		_ = os.Remove(resolvedConf)
		_ = exec.Command("systemctl", "restart", "systemd-resolved").Run()
		fmt.Println("已移除 systemd-resolved 的 freecat DNS 配置")
		return
	}
	b, err := os.ReadFile(resolvBak)
	if err != nil {
		fatal("没有找到备份 %s，请手动恢复 DNS", resolvBak)
	}
	_ = os.Remove(resolvConf)
	if err := os.WriteFile(resolvConf, b, 0o644); err != nil {
		fatal("恢复失败: %v", err)
	}
	_ = os.Remove(resolvBak)
	fmt.Println("已恢复 /etc/resolv.conf")
}

func dnsStatus() {
	if _, err := os.Stat(resolvedConf); err == nil {
		fmt.Println("systemd-resolved: 已配置 DNS=127.0.0.1")
		return
	}
	b, err := os.ReadFile(resolvConf)
	if err != nil {
		fmt.Println("读取 /etc/resolv.conf 失败:", err)
		return
	}
	fmt.Println("当前 /etc/resolv.conf:")
	for i, ln := range strings.Split(string(b), "\n") {
		if i > 4 {
			break
		}
		fmt.Println(" ", ln)
	}
	if strings.Contains(string(b), "nameserver 127.0.0.1") {
		fmt.Println("状态: 已指向 freecat 内核 DNS")
	} else {
		fmt.Println("状态: 未指向 127.0.0.1（TUN 模式建议 sudo freecat dns on）")
	}
}

func cmdEnv(args []string) {
	st, _ := loadState()
	port := 7890
	if st != nil && st.Port != 0 {
		port = st.Port
	}
	fmt.Printf("export http_proxy=http://127.0.0.1:%d\nexport https_proxy=http://127.0.0.1:%d\nexport all_proxy=socks5://127.0.0.1:%d\n", port, port, port)
	fmt.Printf("export no_proxy=localhost,127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,*.local\n")
}

// ---- helpers ----

func writeRuntime(st *State) error {
	raw, err := os.ReadFile(profilePath())
	if err != nil {
		return err
	}
	patched := patchProfile(string(raw), st.Port, st.Controller, st.Secret, st.endpoint(), st.TUN)
	return writeFile0600(runtimePath(), []byte(patched))
}

// selectNode resolves a node name / 1-based index / "auto" inside the main
// selector group and applies it through the core API.
func selectNode(st *State, want string) (string, error) {
	gname, node, err := pickNode(st, want)
	if err != nil {
		return "", err
	}
	if err := newCtl(st.Controller, st.Secret).selectNode(gname, node); err != nil {
		return "", err
	}
	return node, nil
}

func selectInGroup(st *State, gname, want string) error {
	ctl := newCtl(st.Controller, st.Secret)
	ps, err := waitProxies(ctl, 30)
	if err != nil {
		return err
	}
	g, ok := ps[gname]
	if !ok {
		return fmt.Errorf("策略组 %q 不存在", gname)
	}
	node, err := resolveWant(ps, g, want)
	if err != nil {
		return err
	}
	if err := ctl.selectNode(gname, node); err != nil {
		return err
	}
	fmt.Printf("已将 [%s] 切换到: %s\n", gname, node)
	return nil
}

func pickNode(st *State, want string) (string, string, error) {
	ctl := newCtl(st.Controller, st.Secret)
	ps, err := waitProxies(ctl, 30)
	if err != nil {
		return "", "", err
	}
	gname := pickGroup(ps)
	if gname == "" {
		return "", "", fmt.Errorf("没有找到可切换的策略组")
	}
	node, err := resolveWant(ps, ps[gname], want)
	return gname, node, err
}

func waitProxies(ctl *ctlClient, tries int) (map[string]proxyInfo, error) {
	var lastErr error
	for i := 0; i < tries; i++ {
		ps, err := ctl.proxies()
		if err == nil {
			return ps, nil
		}
		lastErr = err
		time.Sleep(300 * time.Millisecond)
	}
	return nil, lastErr
}

func resolveWant(ps map[string]proxyInfo, g proxyInfo, want string) (string, error) {
	switch want {
	case "auto":
		for _, n := range g.All {
			if ps[n].Type == "URLTest" {
				return n, nil
			}
		}
		for _, n := range g.All {
			if ps[n].Type == "Fallback" {
				return n, nil
			}
		}
		return "", fmt.Errorf("该组没有自动选择节点")
	case "list":
		return "", fmt.Errorf("请用 freecat nodes 查看节点")
	}
	if idx, err := strconv.Atoi(want); err == nil {
		if idx < 1 || idx > len(g.All) {
			return "", fmt.Errorf("序号超出范围（1-%d）", len(g.All))
		}
		return g.All[idx-1], nil
	}
	for _, n := range g.All {
		if n == want {
			return n, nil
		}
	}
	return "", fmt.Errorf("节点 %q 不在组内，用 freecat nodes 查看名称", want)
}

func resolveNode(ps map[string]proxyInfo, name string) string {
	for i := 0; i < 5; i++ {
		p, ok := ps[name]
		if !ok || p.Now == "" || p.Now == name {
			return name
		}
		if _, isGroup := ps[p.Now]; isGroup {
			name = p.Now
			continue
		}
		return p.Now
	}
	return name
}

func groupNames(ps map[string]proxyInfo) []string {
	var names []string
	for n, p := range ps {
		switch p.Type {
		case "Selector", "URLTest", "Fallback", "LoadBalance":
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func pickGroup(ps map[string]proxyInfo) string {
	var names []string
	for n, p := range ps {
		if p.Type == "Selector" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		names = groupNames(ps)
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	for _, n := range names {
		if strings.Contains(n, "Freecat") || strings.Contains(n, "自由猫") || strings.Contains(n, "选择") {
			return n
		}
	}
	return names[0]
}

func printGroup(name string, p proxyInfo, ps map[string]proxyInfo, list bool) {
	head := fmt.Sprintf("\n[%s] (%s) 当前: %s", name, p.Type, p.Now)
	if actual := resolveNode(ps, p.Now); actual != p.Now {
		head += " → " + actual
	}
	fmt.Println(head)
	if p.Type != "Selector" && !list {
		fmt.Printf("  （%d 个候选，用 --group %q --all 展开）\n", len(p.All), name)
		return
	}
	for i, n := range p.All {
		mark := "  "
		if n == p.Now {
			mark = "* "
		}
		fmt.Printf("  %s%3d. %s\n", mark, i+1, n)
	}
}

func chownToSudo(paths ...string) {
	if os.Geteuid() != 0 {
		return
	}
	uid, err1 := strconv.Atoi(os.Getenv("SUDO_UID"))
	gid, err2 := strconv.Atoi(os.Getenv("SUDO_GID"))
	if err1 != nil || err2 != nil {
		return
	}
	for _, p := range paths {
		_ = os.Chown(p, uid, gid)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	_ = os.Chmod(dst, 0o600)
	return nil
}

func ensureGeo(corePath string) {
	dir := coreDir()
	exeDir := filepath.Dir(corePath)
	for _, name := range []string{"geoip.metadb", "geosite.dat"} {
		dst := filepath.Join(dir, name)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		candidates := []string{
			filepath.Join(exeDir, "geo", name),
			filepath.Join("/opt/freecat/geo", name),
			filepath.Join(exeDir, name),
		}
		for _, src := range candidates {
			if copyFile(src, dst) == nil {
				break
			}
		}
	}
}

func findCore(explicit string) (string, error) {
	cands := []string{}
	if explicit != "" {
		cands = append(cands, explicit)
	}
	if v := os.Getenv("FREECAT_CORE"); v != "" {
		cands = append(cands, v)
	}
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	cands = append(cands,
		filepath.Join(exeDir, "mihomo"),
		filepath.Join(exeDir, "core", "mihomo"),
		"/opt/freecat/mihomo",
		"/usr/local/bin/mihomo",
	)
	if p, err := exec.LookPath("mihomo"); err == nil {
		cands = append(cands, p)
	}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("找不到 mihomo 内核。请把 mihomo 放到 %s 或 /opt/freecat/mihomo，或用 --core 指定", filepath.Join(exeDir, "mihomo"))
}

func readLine(r *bufio.Reader) string {
	s, _ := r.ReadString('\n')
	return strings.TrimRight(s, "\r\n")
}

func maskEmail(e string) string {
	at := strings.IndexByte(e, '@')
	if at <= 1 {
		return e
	}
	return e[:1] + "***" + e[at:]
}

func randomSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isRoot() bool { return os.Geteuid() == 0 }

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
