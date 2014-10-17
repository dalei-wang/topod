package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"flag"

	"github.com/BurntSushi/toml"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("config")
var format = "%{color}%{time:2006-01-02 15:04:05.000000} ▶ %{level:.3s} %{id:03x}%{color:reset} %{message}"

var (
	configFile         = ""
	defaultConfigFile  = "/etc/topod/topod.toml"
	configDir          = ""
	defaultConfigDir   = "/etc/topod/conf.d/"
	templateDir        = ""
	defaultTemplateDir = "/etc/topod/templates/"

	store    string
	nodes    []string
	watch    bool
	interval int
	prefix   string
	debug    bool
	verbose  bool
	daemon   bool
)

type Config struct {
	Store      string   `toml:"store"`
	StoreNodes []string `toml:"nodes"`
	ConfDir    string   `toml:"confdir"`
	Debug      bool     `toml:"debug"`
	Prefix     string   `toml:"prefix"`
	Watch      bool     `toml:"watch"`
	Interval   int      `toml:"interval"`
	Daemon     bool     `toml:"daemon"`
	Verbose    bool     `toml:"verbose"`
}

func init() {
	flag.StringVar(&store, "store", "etcd", "conf store to use")
	flag.Var(&nodes, "nodes", "storage nodes format, host:port, host:port")
	flag.StringVar(&configFile, "config", "/etc/topod/topod.toml", "config file path")
	flag.StringVar(&configDir, "confdir", "/etc/topod/conf.d/", "topod config dirrectory")

}
