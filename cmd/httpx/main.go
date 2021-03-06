package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/core"
	"github.com/bufsnake/httpx/pkg/log"
	"net/url"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func init() {
	// 开启多核模式
	runtime.GOMAXPROCS(runtime.NumCPU() * 3 / 4)
	// 关闭 GIN Debug模式
	// 设置工具可打开的文件描述符
	var rLimit syscall.Rlimit
	rLimit.Max = 999999
	rLimit.Cur = 999999
	if runtime.GOOS == "darwin" {
		rLimit.Cur = 10240
	}
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_ = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)

	images := "./.images/"
	if !exists(images) {
		err = os.Mkdir(images, 0777)
		if err != nil {
			fmt.Println("create output file path error", err)
			os.Exit(1)
		}
	}

}

func main() {
	conf := config.Terminal{}
	flag.StringVar(&conf.Target, "target", "", "target ip:port/scheme://ip:port")
	flag.StringVar(&conf.Targets, "targets", "", "target ip:port/scheme://ip:port list file")
	flag.IntVar(&conf.Threads, "thread", 10, "config probe thread")
	flag.StringVar(&conf.Proxy, "proxy", "", "config probe proxy, example: http://127.0.0.1:8080")
	flag.IntVar(&conf.Timeout, "timeout", 10, "config probe http request timeout")
	flag.StringVar(&conf.Output, "output", time.Now().Format("200601021504")+".html", "output file name")
	flag.StringVar(&conf.URI, "uri", "", "specify uri for probe or screenshot")
	flag.StringVar(&conf.ChromePath, "chrome-path", "", "chrome browser path")
	flag.StringVar(&conf.HeadlessProxy, "headless-proxy", "", "chrome browser proxy")
	flag.StringVar(&conf.Search, "search", "", "search string from response")
	flag.BoolVar(&conf.DisableScreenshot, "disable-screenshot", false, "disable screenshot")
	flag.BoolVar(&conf.DisplayError, "display-error", false, "display error")
	flag.BoolVar(&conf.AllowJump, "allow-jump", false, "allow jump")
	flag.BoolVar(&conf.Silent, "silent", false, "silent output")
	flag.Parse()
	if conf.Proxy != "" {
		_, err := url.Parse(conf.Proxy)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	probes := make(map[string]bool)
	if conf.Target != "" {
		probes[conf.Target] = true
	} else if conf.Targets != "" {
		file, err := os.ReadFile(conf.Targets)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		urls := strings.Split(string(file), "\n")
		for i := 0; i < len(urls); i++ {
			urls[i] = strings.Trim(urls[i], "\r")
			if urls[i] == "" {
				continue
			}
			probes[urls[i]] = true
		}
	} else {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			probes[sc.Text()] = true
		}
		if err := sc.Err(); err != nil {
			fmt.Println("failed to read input:", err)
			os.Exit(1)
		}
	}
	if len(probes) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	temp_probes := probes
	for probe, _ := range temp_probes {
		temp := &url.URL{}
		var err error
		if strings.HasPrefix(probe, "http://") || strings.HasPrefix(probe, "https://") {
			temp, err = url.Parse(probe)
		} else {
			temp, err = url.Parse("http://" + probe)
		}
		delete(probes, probe)
		if err != nil {
			fmt.Println(probe, err)
			continue
		}
		probes[temp.Host] = true
	}
	if len(probes) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	var p float64 = 0
	once := true
	log := log.Log{
		Percentage: &p,
		AllHTTP:    float64(len(probes) * 2),
		Conf:       conf,
		Once:       &once,
		StartTime:  time.Now(),
		Silent:     conf.Silent,
	}
	go log.Bar()
	conf.Probes = probes
	newCore := core.NewCore(log, conf)
	err := newCore.Probe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if log.Silent {
		return
	}
	fmt.Println()
}

func exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
