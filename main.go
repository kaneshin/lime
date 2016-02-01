package main

import (
	"errors"
	"regexp"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/codegangsta/gin/lib"

	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var (
	buildTime = time.Now()
	logger    = log.New(os.Stdout, "[lime] ", 0)
	immediate = false
)

func main() {
	app := cli.NewApp()
	app.Name = "lime"
	app.Usage = "A live reload utility for Go web applications."
	app.Action = mainAction
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "port",
			Usage: "port for the proxy server",
		},
		cli.IntFlag{
			Name:  "app-port",
			Usage: "port for the Go web server",
		},
		cli.StringFlag{
			Name:  "bin,b",
			Value: "lime-bin",
			Usage: "path of generated binary file",
		},
		cli.StringFlag{
			Name:  "ignore-pattern",
			Usage: "pattern to ignore",
		},
		cli.StringFlag{
			Name:  "build-pattern",
			Value: "(\\.go)",
			Usage: "pattern to build",
		},
		cli.StringFlag{
			Name:  "run-pattern",
			Value: "(\\.html|\\.css|\\.js)",
			Usage: "pattern to run",
		},
		cli.StringFlag{
			Name:  "path,t",
			Value: ".",
			Usage: "path to watch files from",
		},
		cli.BoolFlag{
			Name:  "immediate,i",
			Usage: "run the server immediately after it's built",
		},
		cli.BoolFlag{
			Name:  "godep,g",
			Usage: "use godep when building",
		},
	}

	app.Run(os.Args)
}

var (
	ipat *regexp.Regexp
	bpat *regexp.Regexp
	rpat *regexp.Regexp
)

func mainAction(c *cli.Context) {
	immediate = c.GlobalBool("immediate")

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}

	bp := wd
	args := c.Args()
	if len(args) > 0 {
		bp = args[0]
	}
	builder := gin.NewBuilder(bp, c.GlobalString("bin"), c.GlobalBool("godep"))
	var runner gin.Runner
	if len(args) < 2 {
		runner = gin.NewRunner(builder.Binary())
	} else {
		runner = gin.NewRunner(builder.Binary(), args[1:]...)
	}
	runner.SetWriter(os.Stdout)
	proxy := gin.NewProxy(builder, runner)

	if port := c.GlobalInt("port"); port > 0 {
		appPort := c.GlobalInt("app-port")
		if appPort == 0 {
			appPort = port + 1
		}
		config := &gin.Config{
			Port:    port,
			ProxyTo: "http://localhost:" + strconv.Itoa(appPort),
		}

		if err := proxy.Run(config); err != nil {
			logger.Fatal(err)
		}

		logger.Printf("listening on port %d\n", port)
	}

	shutdown(runner)

	// build right now
	build(builder, runner, logger)

	// scan for changes
	if p := c.GlobalString("ignore-pattern"); len(p) > 0 {
		ipat = regexp.MustCompile(p)
	}
	if p := c.GlobalString("build-pattern"); len(p) > 0 {
		bpat = regexp.MustCompile(p)
	}
	if p := c.GlobalString("run-pattern"); len(p) > 0 {
		rpat = regexp.MustCompile(p)
	}

	targets := strings.Split(c.GlobalString("path"), ",")
	for {
		for _, target := range targets {
			scanChanges(filepath.Clean(filepath.Join(wd, target)), func(path string) error {
				ext := filepath.Ext(path)
				switch {
				case bpat != nil && bpat.MatchString(ext):
					logger.Printf("Detected file changes: %s", path)
					buildTime = time.Now()
					runner.Kill()
					build(builder, runner, logger)
				case rpat != nil && rpat.MatchString(ext):
					logger.Printf("Detected file changes: %s", path)
					buildTime = time.Now()
					runner.Kill()
					runner.Run()
				default:
					return nil
				}
				return errors.New("done")
			})
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func build(builder gin.Builder, runner gin.Runner, logger *log.Logger) {
	logger.Println("Build started.")
	if err := builder.Build(); err != nil {
		logger.Println("ERROR! Build failed.")
		logger.Println(builder.Errors())
		re := regexp.MustCompile("cannot find package \".*\"")
		matches := re.FindAllStringSubmatch(builder.Errors(), -1)
		goget(matches)
	} else {
		logger.Println("Build Successful.")
		if immediate {
			runner.Run()
		}
	}

	time.Sleep(100 * time.Millisecond)
}

func goget(packs [][]string) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		logger.Fatalf("Go executable not found in PATH.")
	}
	for _, pack := range packs {
		for _, p := range pack {
			rep := strings.Replace(strings.Replace(p, "cannot find package ", "", -1), `"`, "", -1)
			args := []string{"get", "-u", rep}
			cmd := exec.Command(goPath, args...)
			logger.Printf("go get -u %s\n", rep)
			cmd.CombinedOutput()
		}
	}
}

type scanCallback func(path string) error

func scanChanges(watchPath string, cb scanCallback) {
	filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Fatal(err)
			return nil
		}

		if ipat != nil && ipat.MatchString(path) {
			return filepath.SkipDir
		}

		if info.ModTime().After(buildTime) {
			return cb(path)
		}

		return nil
	})
}

func shutdown(runner gin.Runner) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		logger.Println("Got signal: ", s)
		err := runner.Kill()
		if err != nil {
			logger.Print("Error killing: ", err)
		}
		os.Exit(1)
	}()
}
