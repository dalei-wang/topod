package template

import (
	"bytes"
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/op/go-logging"
	"github.com/wlsailor/topod/store"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

//Template global config , value init from topod config
type Config struct {
	ParentDir    string
	ConfDir      string
	TemplateDir  string
	Prefix       string
	Noop         bool
	StoreClient  store.StoreClient
	KeepTempFile bool
}

//Template file parsed config, part of vars from template global config above, some from xxx_xxx.toml config file
type TemplateResource struct {
	CheckCmd     string `toml:"check_cmd"`
	Dest         string `toml:"dest"`
	FileMode     os.FileMode
	Gid          int
	Keys         []string `toml:"keys"`
	Mode         string   `toml:"mode"`
	Prefix       string   `toml:"prefix"`
	ReloadCmd    string   `toml:"reload_cmd"`
	Src          string   `toml:"src"`
	TempFile     *os.File
	Backup       bool   `toml:"backup"`
	BackupDir    string `toml:"backupdir"`
	Uid          int
	funcMap      map[string]interface{}
	cache        map[string]string
	lastIndex    uint64
	noop         bool
	storeClient  store.StoreClient
	keepTempFile bool
}

var log = logging.MustGetLogger("template")
var format = "%{color}%{time:2006-01-02 15:04:05.000000} > %{level:.3s} %{id:03x}%{color:reset} %{message}"
var emptySrcError = errors.New("empty template src path error")

func init() {
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logging.SetBackend(logBackend)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetLevel(logging.DEBUG, "templte")
}

func NewConfigTemplate(path string, config *Config) (*TemplateResource, err) {
	if config.storeClient == nil {
		return nil, errors.New("A valid store client required")
	}
	if path == "" {
		return nil, errors.New("Empty config template file path")
	}
	log.Debug("Loading template resource path %s", path)
	var tr *TemplateResource
	_, err := toml.DecodeFile(path, tr)
	tr.storeClient = config.StoreClient
	tr.noop = config.Noop
	tr.keepTempFile = config.KeepTempFile
	tr.funcMap = newFuncMap()
	tr.cache = make(map[string]string)
	tr.Prefix = filepath.Join("/", config.Prefix, tr.Prefix)
	if tr.Backup && tr.BackupDir == "" {
		tr.BackupDir = tr.Dest
	}
	if tr.Src == "" {
		return nil, emptySrcError
	}
	tr.Src = filepath.Join("/", config.TemplateDir, tr.Src)
	return tr, nil
}

func getTemplateResource(config *Config) ([]*TemplateResource, error) {
	var lastError error
	templates := make([]*TemplateResource, 0)
	log.Debug("Loading template resources from conf dir %s", config.ConfDir)
	if !isFileExist(config.ConfDir) {
		log.Warning("Conf dir %s does not exist", config.ConfDir)
		return nil, errors.New("Conf dir does not exist")
	}
	paths, err := filepath.Glob(filepath.Join(config.ConfDir, "*.toml"))
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		template, err := NewConfigTemplate(path, config)
		if err != nil {
			lastError = err
			continue
		}
		templates = append(templates, template)
	}
	return templates, lastError
}

func (t *TemplateResource) setVars() error {
	var err error
	log.Debug("Retrieving keys from store, key prefix:%s", t.Prefix)
	result, err := t.storeClient.GetValues(appendPrefixKeys(t.Prefix, t.Keys))
	if err != nil {
		return err
	}
	for k, v := range result {
		t.cache[k] = v
	}
	return nil
}

func (t *TemplateResource) createTempFile() error {
	log.Debug("Loading source template %s", t.Src)
	if !isFileExist(t.Src) {
		return errors.New("Missing template " + t.Src)
	}
	//create template config file in dest dir
	temp, err := ioutil.TempFile(t.Dest, "."+filepath.Base(t.Dest))
	if err != nil {
		log.Error("create temp file error: %s", err.Error())
		return err
	}
	//set owner group mode to the temp file
	os.Chmod(temp.Name(), t.FileMode)
	os.Chown(temp.Name(), t.Uid, t.Gid)
	t.TempFile = temp
	return nil
}

func (t *TemplateResource) setFileMode() error {
	if t.Mode == "" {
		if !isFileExist(t.Dest) {
			t.FileMode = 0644
		} else {
			fi, err := os.Stat(t.Dest)
			if err != nil {
				return err
			}
			t.FileMode = fi.Mode()
		}
	} else {
		mode, err := strconv.ParseUint(t.Mode, 0, 32)
		if err != nil {
			return err
		}
		t.FileMode = mode
	}
}

// check executes the check command to validate the staged config file. The
// command is modified so that any references to src template are substituted
// with a string representing the full path of the staged file. This allows the
// check to be run on the staged file before overwriting the destination config
// file.
// It returns nil if the check command returns 0 and there are no other errors.
func (t *TemplateResource) check() error {
	var cmdBuffer bytes.Buffer
	data := make(map[string]string)
	data["src"] = t.TempFile.Name()
	tmpl, err := template.New("checkcmd").Parse(t.CheckCmd)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(&cmdBuffer, data); err != nil {
		return err
	}
	log.Debug("Running " + cmdBuffer.String())
	c := exec.Command("/bin/sh", "-c", cmdBuffer.String())
	output, err := c.CombinedOutput()
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("%q", string(output)))
	return nil
}

// reload executes the reload command.
// It returns nil if the reload command returns 0.
func (t *TemplateResource) reload() error {
	log.Debug("Running " + t.ReloadCmd)
	c := exec.Command("/bin/sh", "-c", t.ReloadCmd)
	output, err := c.CombinedOutput()
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("%q", string(output)))
	return nil
}

func (t *TemplateResource) sync() error {
	temp := t.TempFile.Name()
	if t.keepTempFile {
		log.Info("Keeping temp config file: %s", temp)
	} else {
		defer os.Remove(temp)
	}
	log.Debug("Comparing candidate config to %s", t.Dest)
	//TODO check if the same
}

func (t *TemplateResource) process() error {
	if err := t.setFileMode(); err != nil {
		return err
	}
	if err := t.setVars(); err != nil {
		return err
	}
	if err := t.createTempFile(); err != nil {
		return err
	}
	if err := t.sync(); err != nil {
		return err
	}
	return nil
}
