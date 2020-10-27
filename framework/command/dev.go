package command

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"hade/framework/cobra"
	commandUtil "hade/framework/command/util"
	"hade/framework/contract"
	"hade/framework/util"

	"github.com/fsnotify/fsnotify"
)

const (
	frontendPrefix = "/dist"
	swaggerPrefix  = "/swagger"
)

var (
	openSwagger bool
	refreshTime int32
	proxyURL    string
)

// Proxy 代表serve启动的backend的服务器代理
type Proxy struct {
	// proxy信息
	proxyURL     *url.URL
	proxyServer  *http.Server
	proxyReverse *httputil.ReverseProxy

	url    *url.URL // 最终轮询的地址
	server *http.Server

	backendPid  int // backend服务的pid
	frontendPid int
	swaggerPid  int

	hander func(res http.ResponseWriter, req *http.Request)

	lock *sync.RWMutex
}

// 重新启动一个proxy网关
func newProxyReverseProxy(backend, frontend, swagger *url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = backend.Scheme
		req.URL.Host = backend.Host
		if frontend != nil && strings.HasPrefix(req.URL.Path, frontendPrefix) {
			req.URL.Scheme = frontend.Scheme
			req.URL.Host = frontend.Host
			req.URL.Path = strings.ReplaceAll(req.URL.Path, frontendPrefix, "")
		}
		if swagger != nil && strings.HasPrefix(req.URL.Path, swaggerPrefix) {
			req.URL.Scheme = swagger.Scheme
			req.URL.Host = swagger.Host
			// req.URL.Path = strings.ReplaceAll(req.URL.Path, swaggerPrefix, "")
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

func (p *Proxy) rebuildBackend() error {
	// 重新编译hade
	cmdBuild := exec.Command("./hade", "build", "self")
	cmdBuild.Stdout = os.Stdout
	cmdBuild.Stderr = os.Stderr
	if err := cmdBuild.Start(); err == nil {
		err = cmdBuild.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) rebuildSwagger() error {
	// 重新编译hade
	cmdBuild := exec.Command("./hade", "swagger", "gen")
	cmdBuild.Stdout = os.Stdout
	cmdBuild.Stderr = os.Stderr
	if err := cmdBuild.Start(); err == nil {
		err = cmdBuild.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

// 启动swagger服务
func (p *Proxy) startSwagger() (swaggerUrl *url.URL, pid int, err error) {
	swaggerAddress := fmt.Sprintf("http://%s:%d", "127.0.0.1", 8080)
	swaggerUrl, err = url.Parse(swaggerAddress)
	cmd := exec.Command("./hade", "swagger", "serve", "--address="+swaggerAddress)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return
	}
	pid = cmd.Process.Pid
	fmt.Println("swagger server:", swaggerAddress)
	return
}

// 启动后端服务
func (p *Proxy) startBackend() (backendUrl *url.URL, pid int, err error) {
	// 设置随机端口，真实后端的端口
	rand.Seed(time.Now().UnixNano())
	port := rand.Int63n(10000) + 10000
	hadeAddress := fmt.Sprintf("http://%s:%d", "127.0.0.1", port)
	backendUrl, err = url.Parse(hadeAddress)
	if err != nil {
		return
	}
	// 使用命令行启动后端进程
	cmd := exec.Command("./hade", "app", "start", "--address="+hadeAddress)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return
	}
	pid = cmd.Process.Pid
	fmt.Println("backend server:", hadeAddress)

	return
}

// 启动前端服务
func (p *Proxy) startFrontend() (frontendUrl *url.URL, pid int, err error) {
	// 启动前端调试模式
	rand.Seed(time.Now().UnixNano())
	port := rand.Int63n(10000) + 10000
	path, err := exec.LookPath("npm")
	if err != nil {
		return
	}
	cmd := exec.Command(path, "run", "serve")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s%d", "PORT=", port))
	cmd.Stdout = os.NewFile(0, os.DevNull)
	cmd.Stderr = os.Stderr
	pid = cmd.Process.Pid
	go func() {
		err := cmd.Run()
		fmt.Println("frontend server: ", "http://127.0.0.1:", port)
		if err != nil {
			fmt.Println(err)
		}
	}()

	frontendUrl, err = url.Parse(fmt.Sprintf("%s%d", "http://127.0.0.1:", port))
	if err != nil {
		return
	}
	return
}

// 重启后端服务, 如果frontend为nil，则没有包含后端
func (p *Proxy) startProxy(startBackend, startFrontend, startSwagger bool) error {
	var err error
	p.lock.Lock()
	defer p.lock.Unlock()
	var backendURL, frontendURL, swaggerURL *url.URL
	var backendPid, frontendPid, swaggerPid int

	if startBackend {
		if backendURL, backendPid, err = p.startBackend(); err != nil {
			return err
		}
	}
	if startFrontend {
		if frontendURL, frontendPid, err = p.startFrontend(); err != nil {
			return nil
		}
	}
	if startSwagger {
		p.rebuildSwagger()
		if swaggerURL, swaggerPid, err = p.startSwagger(); err != nil {
			return nil
		}
	}

	if p.proxyServer != nil {
		p.proxyServer.Close()
	}

	p.proxyReverse = newProxyReverseProxy(backendURL, frontendURL, swaggerURL)
	p.proxyServer = &http.Server{
		Addr:    "127.0.0.1:" + p.proxyURL.Port(),
		Handler: p.proxyReverse,
	}
	go func() {
		err := p.proxyServer.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()
	fmt.Println("proxy backend server:", p.proxyURL.String())

	if p.backendPid != 0 {
		syscall.Kill(p.backendPid, syscall.SIGKILL)
		p.backendPid = backendPid
	}
	if p.frontendPid != 0 {
		syscall.Kill(p.frontendPid, syscall.SIGKILL)
		p.frontendPid = frontendPid
	}
	if p.swaggerPid != 0 {
		syscall.Kill(p.swaggerPid, syscall.SIGKILL)
		p.swaggerPid = swaggerPid
	}

	return nil
}

// 初始化Dev命令
func initDevCommand() *cobra.Command {
	devCommand.PersistentFlags().BoolVarP(&openSwagger, "swagger", "s", false, "是否打开swagger")
	devCommand.PersistentFlags().Int32VarP(&refreshTime, "refresh", "r", 3, "更新时间")
	devCommand.PersistentFlags().StringVarP(&proxyURL, "address", "a", "http://127.0.0.1:8066", "")

	devCommand.AddCommand(devBackendCommand)
	devCommand.AddCommand(devFrontendCommand)
	devCommand.AddCommand(devAllCommand)
	return devCommand
}

var devCommand = &cobra.Command{
	Use:   "dev",
	Short: "dev mode",
	RunE: func(c *cobra.Command, args []string) error {
		c.Help()
		return nil
	},
}

// serveCommand start a app serve
var devBackendCommand = &cobra.Command{
	Use:   "backend",
	Short: "dev mode for backend, hot reload",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		proxyURL, err := url.Parse(proxyURL)
		if err != nil {
			return err
		}
		proxy := &Proxy{proxyURL: proxyURL, lock: &sync.RWMutex{}}

		if err = proxy.rebuildBackend(); err != nil {
			return err
		}

		if err = proxy.startProxy(true, false, openSwagger); err != nil {
			return err
		}

		// 监听
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		defer watcher.Close()

		appFolder := appService.AppPath()
		swaggerFolder := appService.SwaggerPath()
		filepath.Walk(appFolder, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() {
				return nil
			}
			if util.IsHiddenDirectory(path) {
				return nil
			}
			if !info.IsDir() && (filepath.Dir(path) == filepath.Dir(swaggerFolder)) {
				return nil
			}
			if info.IsDir() && (path == filepath.Dir(swaggerFolder)) {
				return nil
			}

			return watcher.Add(path)
		})

		t := time.NewTimer(time.Duration(refreshTime) * time.Second)
		t.Stop()
		for {
			select {
			case <-t.C:
				fmt.Println("...detect some file change, hot reload start...")
				proxy.rebuildBackend()
				if openSwagger {
					proxy.rebuildSwagger()
				}
				proxy.startProxy(true, false, openSwagger)
				fmt.Println("...detect some file change, hot reload finish...")
				t.Stop()
			case ev, ok := <-watcher.Events:
				if !ok {
					continue
				}
				fmt.Println(ev)
				t.Reset(time.Duration(refreshTime) * time.Second)
			case err, ok := <-watcher.Errors:
				if !ok {
					continue
				}
				log.Panicln(err)
				t.Reset(time.Duration(refreshTime) * time.Second)
			}
		}
	},
}

var devFrontendCommand = &cobra.Command{
	Use:   "frontend",
	Short: "dev mode for frontend",
	RunE: func(c *cobra.Command, args []string) error {
		path, err := exec.LookPath("npm")
		if err != nil {
			log.Fatalln("hade npm: should install npm in your PATH")
			return err
		}

		cmd := exec.Command(path, "run", "serve")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		return nil
	},
}

var devAllCommand = &cobra.Command{
	Use:   "all",
	Short: "dev mode from both frontend and backend",
	RunE: func(c *cobra.Command, args []string) error {
		container := commandUtil.GetContainer(c.Root())
		appService := container.MustMake(contract.AppKey).(contract.App)

		proxyURL, err := url.Parse(proxyURL)
		if err != nil {
			return err
		}
		proxy := &Proxy{proxyURL: proxyURL, lock: &sync.RWMutex{}}

		if err = proxy.rebuildBackend(); err != nil {
			return err
		}

		if err = proxy.startProxy(true, false, openSwagger); err != nil {
			return err
		}

		// 监听
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		defer watcher.Close()

		appFolder := appService.AppPath()
		swaggerFolder := appService.SwaggerPath()
		filepath.Walk(appFolder, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() {
				return nil
			}
			if util.IsHiddenDirectory(path) {
				return nil
			}
			if !info.IsDir() && (filepath.Dir(path) == filepath.Dir(swaggerFolder)) {
				return nil
			}
			if info.IsDir() && (path == filepath.Dir(swaggerFolder)) {
				return nil
			}

			return watcher.Add(path)
		})

		t := time.NewTimer(time.Duration(refreshTime) * time.Second)
		t.Stop()
		for {
			select {
			case <-t.C:
				fmt.Println("...detect some file change, hot reload start...")
				proxy.rebuildBackend()
				if openSwagger {
					proxy.rebuildSwagger()
				}
				proxy.startProxy(true, true, openSwagger)
				fmt.Println("...detect some file change, hot reload finish...")
				t.Stop()
			case ev, ok := <-watcher.Events:
				if !ok {
					continue
				}
				fmt.Println(ev)
				t.Reset(time.Duration(refreshTime) * time.Second)
			case err, ok := <-watcher.Errors:
				if !ok {
					continue
				}
				log.Panicln(err)
				t.Reset(time.Duration(refreshTime) * time.Second)
			}
		}
	},
}
