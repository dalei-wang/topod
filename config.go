package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	//"strconv"
	//"strings"

	"github.com/BurntSushi/toml"

	"github.com/wlsailor/topod/conf/template"
	storage "github.com/wlsailor/topod/store"
)

var (
	configFile         = ""
	defaultConfigFile  = "/etc/topod/topod.toml"
	configDir          = ""
	defaultConfigDir   = "/etc/topod/conf.d/"
	templateDir        = ""
	defaultTemplateDir = "/etc/topod/templates/"

	store    string
	nodes    Nodes
	schema   string
	watch    bool
	interval int
	prefix   string
	debug    bool
	verbose  bool
	noop     bool
	daemon   bool
	version  bool

	//hold global config
	config         Config
	storeConfig    storage.Config
	templateConfig template.Config
)

//Nodes alias []string used for flag value init, []string is forbiden for it does not contain Set method
type Nodes []string

func (n *Nodes) String() string {
	return fmt.Sprintf("%s", *n)
}
func (n *Nodes) Set(node string) error {
	*n = append(*n, node)
	return nil
}

type Config struct {
	Store      string   `toml:"store"`
	StoreNodes []string `toml:"nodes"`
	Schema     string   `toma:"schema"`
	ConfDir    string   `toml:"confdir"`
	Debug      bool     `toml:"debug"`
	Prefix     string   `toml:"prefix"`
	Watch      bool     `toml:"watch"`
	Interval   int      `toml:"interval"`
	Daemon     bool     `toml:"daemon"`
	Verbose    bool     `toml:"verbose"`
	Noop       bool     `toml:"noop"`
}

func init() {
	flag.StringVar(&store, "store", "etcd", "conf store to use")
	flag.Var(&nodes, "nodes", "storage nodes format, host:port, host:port")
	flag.StringVar(&schema, "schema", "http", "the store service uri schema(http|https)")
	flag.StringVar(&configFile, "config", "/etc/topod/topod.toml", "config file path")
	flag.StringVar(&configDir, "confdir", "/etc/topod/conf.d/", "topod config dirrectory")
	flag.StringVar(&prefix, "prefix", "/", "key path prefix")
	flag.BoolVar(&debug, "debug", false, "whether to enable debug log level")
	flag.BoolVar(&watch, "watch", false, "use watch mode or pull mode,  if false, interval config is valid")
	flag.IntVar(&interval, "interval", 60, "pull config interval in secondes")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose log level")
	flag.BoolVar(&daemon, "daemon", false, "process keep alive, not once and exit")
	flag.BoolVar(&noop, "noop", false, "only show pending changes")
	flag.BoolVar(&version, "version", false, "print version and exit")
}

/*
* First init default config then override from config file and then overriding from flag set on the command line.
 */
func initConfig() error {
	if configFile == "" {
		if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
			configFile = defaultConfigFile
		}
	}
	//init config struct
	config = Config{
		Store:      "etcd",
		StoreNodes: []string{"127.0.0.1:4001"},
		Schema:     "http",
		ConfDir:    defaultConfigDir,
		Prefix:     "/",
		Interval:   60,
	}
	//update config from config file
	if configFile == "" {
		log.Warning("Skiping config file, file not specified")
	} else {
		log.Debug("Start loading config file " + configFile)
		configBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return err
		}
		_, err = toml.Decode(string(configBytes), &config)
		if err != nil {
			return err
		}
	}
	//update config from command line
	processFlags()
	//get store service uri from some other service
	//TODO

	if len(config.StoreNodes) == 0 {
		switch config.Store {
		case "consule":
			config.StoreNodes = []string{"127.0.0.1:8500"}
		case "etcd":
			config.StoreNodes = []string{"127.0.0.1:4001"}
		}
	}

	storeConfig = storage.Config{
		Store:  config.Store,
		Nodes:  config.StoreNodes,
		Schema: config.Schema,
	}
	templateConfig = template.Config{
		ParentDir:   config.ConfDir,
		ConfDir:     filepath.Join(config.ConfDir, "conf.d"),
		TemplateDir: filepath.Join(config.ConfDir, "templates"),
		Prefix:      config.Prefix,
		Noop:        config.Noop,
	}
	return nil
}

func processFlags() {
	flag.Visit(updateConfigFromFlag)
}

func updateConfigFromFlag(f *flag.Flag) {
	switch f.Name {
	case "store":
		config.Store = store
	case "nodes":
		config.StoreNodes = nodes
	case "schema":
		config.Schema = schema
	case "confdir":
		config.ConfDir = configDir
	case "watch":
		config.Watch = watch
	case "interval":
		config.Interval = interval
	case "debug":
		config.Debug = debug
	case "verbose":
		config.Verbose = verbose
	case "daemon":
		config.Daemon = daemon
	case "prefix":
		config.Prefix = prefix
	case "noop":
		config.Noop = noop
	}
}
