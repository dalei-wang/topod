package main

import (
	//"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	//"strconv"
	//"strings"

	"github.com/BurntSushi/toml"
	"github.com/voxelbrain/goptions"

	"github.com/leightonwong/topod/conf/template"
	"github.com/leightonwong/topod/logger"
	storage "github.com/leightonwong/topod/store"
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
	options        CommandOptions
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

//Use goptions instead of flag to parse command
type WatchOptions struct {
	Once bool `goptions:"-o, --once, description='watch once , when changed regenerate config file and exit'"`
}
type PullOptions struct {
	Interval int `goptions:"-i, --interval, obligatory, description='pull config from remote server, in seconds'"`
}
type GenOptions struct {
}
type CommandOptions struct {
	Store      string `goptions:"-s, --store, description='remote conf store to use, etcd or consule'"`
	StoreNodes Nodes  `goptions:"-N, --nodes, description='remote storage uri, format host:port, host:port'"`
	Schema     string `goptions:"-m, --schema, description='remote storage service schema(http|https)'"`
	Config     string `goptions:"-c, --config, description='topod config file path'"`
	ConfDir    string `goptions:"-d, --confdir, description='topod config directory'"`
	Prefix     string `goptions:"-p, --prefix, description='key path prefix'"`
	//Backup     bool          `goptions:"-b, --backup, description='enable backup config file'"`
	//BackupDir  string        `goptions:"-bd, --backupdir, description='back up directories, default use config file current dir'"`
	Debug   bool          `goptions:"-D, --debug, description='enable debug log level'"`
	Verbose bool          `goptions:"-v, --verbose, description='enable verbose log level'"`
	Noop    bool          `goptions:"-n, --noop, description='only show pending changes'"`
	Version bool          `goptions:"-V, --version, description='print version and exit'"`
	Help    goptions.Help `goptions:"-h, --help, description='show help'"`
	goptions.Verbs
	Watch WatchOptions `goptions:"watch"`
	Pull  PullOptions  `goptions:"pull"`
	Gen   GenOptions   `goptions:"gen"`
}

type Config struct {
	Store      string   `toml:"store"`
	StoreNodes []string `toml:"nodes"`
	Schema     string   `toma:"schema"`
	ConfDir    string   `toml:"confdir"`
	Debug      bool     `toml:"debug"`
	Prefix     string   `toml:"prefix"`
	Watch      WatchOptions
	Pull       PullOptions
	Gen        GenOptions
	Verbose    bool `toml:"verbose"`
	Noop       bool `toml:"noop"`
}

func init() {
	/*
		flag.StringVar(&store, "store", "etcd", "conf store to use")
		flag.Var(&nodes, "nodes", "storage nodes format, host:port, host:port")
		flag.StringVar(&schema, "schema", "http", "the store service uri schema(http|https)")
		flag.StringVar(&configFile, "config", "/etc/topod/topod.toml", "config file path")
		flag.StringVar(&configDir, "confdir", "/etc/topod/conf.d/", "topod config dirrectory")
		flag.StringVar(&prefix, "prefix", "/", "key path prefix")
		flag.BoolVar(&debug, "debug", false, "whether to enable debug logger.Log.level")
		flag.BoolVar(&watch, "watch", false, "use watch mode or pull mode,  if false, interval config is valid")
		flag.IntVar(&interval, "interval", 60, "pull config interval in secondes")
		flag.BoolVar(&verbose, "verbose", false, "enable verbose logger.Log.level")
		flag.BoolVar(&daemon, "daemon", false, "process keep alive, not once and exit")
		flag.BoolVar(&noop, "noop", false, "only show pending changes")
		flag.BoolVar(&version, "version", false, "print version and exit")
	*/
	if len(os.Args) == 2 && os.Args[1] == "-test.v=true" {
		os.Args = os.Args[0:1]
	}
	options = CommandOptions{
	//Store:   "etcd",
	//Schema:  "http",
	//Config:  "/etc/topod/topod.toml",
	//ConfDir: "/etc/topod/conf.d/",
	//Prefix:  "/",
	}
	//logger.Log.Info("Command line args:%v", os.Args)
	goptions.ParseAndFail(&options)
	//logger.Log.Debug("Parsed options verbs: %s", options.Verbs)
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
	}
	//update config from config file
	if configFile == "" {
		logger.Log.Warning("Skiping config file, file not specified")
	} else {
		logger.Log.Debug("Start loading config file " + configFile)
		configBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			//logger.Log.Warning("Reading config file %s error : %s, use empty config instead", configFile, err.Error())
			return err
		}
		_, err = toml.Decode(string(configBytes), &config)
		if err != nil {
			return err
		}
	}
	//update config from command line
	//processFlags()
	processOptions()
	//get store service uri from some other service

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

/*
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
*/
func processOptions() {
	if options.Store != "" {
		config.Store = options.Store
	}
	if options.ConfDir != "" {
		config.ConfDir = options.ConfDir
	}
	if options.Debug {
		config.Debug = options.Debug
	}
	if options.Noop {
		config.Noop = options.Noop
	}
	if options.Prefix != "" {
		config.Prefix = options.Prefix
	}
	if options.Schema != "" {
		config.Schema = options.Schema
	}
	if options.StoreNodes != nil {
		config.StoreNodes = options.StoreNodes
	}
	if options.Verbose != false {
		config.Verbose = options.Verbose
	}
	switch options.Verbs {
	case "watch":
		config.Watch = options.Watch
	case "gen":
		config.Gen = options.Gen
	case "pull":
		config.Pull = options.Pull
	}
}
