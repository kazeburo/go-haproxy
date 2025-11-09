package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/kazeburo/haproxy"
)

var version string
var commit string

type Opt struct {
	Version     bool          `short:"v" long:"version" description:"Show version"`
	HaproxyHost string        `short:"H" long:"haproxy-host" description:"HAProxy host" default:"localhost"`
	HaproxyPort int           `short:"P" long:"haproxy-port" description:"HAProxy port" default:"8080"`
	Timeout     time.Duration `short:"t" long:"timeout" description:"Timeout in seconds" default:"5s"`
}

func main() {
	os.Exit(_main())
}

func _main() int {
	opt := Opt{}
	psr := flags.NewParser(&opt, flags.HelpFlag|flags.PassDoubleDash)
	_, err := psr.Parse()
	if opt.Version {
		if commit == "" {
			commit = "dev"
		}
		fmt.Printf(
			"%s-%s\n%s/%s, %s, %s\n",
			filepath.Base(os.Args[0]),
			version,
			runtime.GOOS,
			runtime.GOARCH,
			runtime.Version(),
			commit)
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	status, err := haproxy.Status(
		haproxy.Host(opt.HaproxyHost),
		haproxy.Port(fmt.Sprintf("%d", opt.HaproxyPort)),
		haproxy.HTTPClient(&http.Client{Timeout: opt.Timeout}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	// JSON format output
	json.NewEncoder(os.Stdout).Encode(status)
	return 0
}
