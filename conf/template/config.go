package template

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"

	"github.com/wlsailor/topod/logger"
	"github.com/wlsailor/topod/memkv"
	"github.com/wlsailor/topod/store"
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
	cache        *memkv.MemStore
	lastIndex    uint64
	noop         bool
	storeClient  store.StoreClient
	keepTempFile bool
}

var EmptySrcErr = errors.New("empty src template")

/*
* New template resource from toml config file in conf.d
 */
func NewConfigTemplate(path string, config *Config) (*TemplateResource, error) {
	if config.StoreClient == nil {
		return nil, errors.New("A valid store client required")
	}
	if path == "" {
		return nil, errors.New("Empty config template file path")
	}
	logger.Log.Debug("Loading template resource path %s", path)
	var tr TemplateResource
	_, err := toml.DecodeFile(path, &tr)
	if err != nil {
		logger.Log.Error("Error decoding toml file %s, error: %s", path, err.Error())
		return nil, err
	}
	tr.storeClient = config.StoreClient
	tr.noop = config.Noop
	tr.keepTempFile = config.KeepTempFile
	tr.funcMap = newFuncMap()
	tr.cache = memkv.NewMemStore()
	addFuncs(tr.funcMap, tr.cache.FuncMap)
	tr.Prefix = filepath.Join("/", config.Prefix, tr.Prefix)
	if tr.Backup && tr.BackupDir == "" {
		tr.BackupDir = filepath.Dir(tr.Dest)
	}
	if tr.Src == "" {
		return nil, EmptySrcErr
	}
	tr.Src = filepath.Join("/", config.TemplateDir, tr.Src)
	return &tr, nil
}

func getTemplateResource(config *Config) ([]*TemplateResource, error) {
	var lastError error
	templates := make([]*TemplateResource, 0)
	logger.Log.Debug("Loading template resources from conf dir %s", config.ConfDir)
	if !isFileExist(config.ConfDir) {
		logger.Log.Warning("Conf dir %s does not exist", config.ConfDir)
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
	logger.Log.Debug("Retrieving keys from store, key prefix:%s", t.Prefix)
	result, err := t.storeClient.GetValues(appendPrefixKeys(t.Prefix, t.Keys))
	if err != nil {
		return err
	}
	t.cache.Clear()
	for k, v := range result {
		t.cache.Set(filepath.Join("/", strings.TrimPrefix(k, t.Prefix)), v)
	}
	return nil
}

func (t *TemplateResource) createTempFile() error {
	logger.Log.Debug("Loading source template %s", t.Src)
	if !isFileExist(t.Src) {
		return errors.New("Missing template " + t.Src)
	}
	//create template config file in dest dir
	temp, err := ioutil.TempFile(filepath.Dir(t.Dest), "."+filepath.Base(t.Dest))
	if err != nil {
		logger.Log.Error("Create temp file error: %s", err.Error())
		return err
	}
	defer temp.Close()
	logger.Log.Debug("Compiling source template %s", t.Src)
	tmpl := template.Must(template.New(path.Base(t.Src)).Funcs(t.funcMap).ParseFiles(t.Src))
	if err = tmpl.Execute(temp, nil); err != nil {
		return err
	}
	//set owner group mode to the temp file
	os.Chmod(temp.Name(), t.FileMode)
	os.Chown(temp.Name(), t.Uid, t.Gid)
	t.TempFile = temp
	logger.Log.Debug("Create temp file %s", temp.Name())
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
		t.FileMode = os.FileMode(mode)
	}
	return nil
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
	logger.Log.Debug("Running " + cmdBuffer.String())
	c := exec.Command("/bin/sh", "-c", cmdBuffer.String())
	output, err := c.CombinedOutput()
	if err != nil {
		return err
	}
	logger.Log.Debug(fmt.Sprintf("%q", string(output)))
	return nil
}

// reload executes the reload command.
// It returns nil if the reload command returns 0.
func (t *TemplateResource) reload() error {
	logger.Log.Debug("Running " + t.ReloadCmd)
	c := exec.Command("/bin/sh", "-c", t.ReloadCmd)
	output, err := c.CombinedOutput()
	if err != nil {
		return err
	}
	logger.Log.Debug(fmt.Sprintf("%q", string(output)))
	return nil
}

func (t *TemplateResource) sync() error {
	temp := t.TempFile.Name()
	if t.keepTempFile {
		logger.Log.Info("Keeping temp config file: %s", temp)
	} else {
		defer os.Remove(temp)
	}
	logger.Log.Debug("Comparing candidate config to %s", t.Dest)
	//check if the same
	result, err := isSameFile(temp, t.Dest)
	if err != nil {
		logger.Log.Error(err.Error())
	}
	if t.noop {
		logger.Log.Warning("Noop mode enabled %s will not be modified", t.Dest)
		return nil
	}
	if !result {
		logger.Log.Info("Target config %s out of sync", t.Dest)
		if t.CheckCmd != "" {
			if err := t.check(); err != nil {
				return errors.New("Config check failed: " + err.Error())
			}
		}
		//Back up original config file
		if t.Backup {
			logger.Log.Debug("Begin to backup config file %s to dir %s", t.Dest, t.BackupDir)
			backup, err := backupFile(t.Dest, t.BackupDir)
			if err != nil {
				logger.Log.Debug("Backup config file %s failed error: %s", t.Dest, err.Error())
			} else {
				logger.Log.Info("Backup config file %s to %s", t.Dest, backup)
			}
		}
		logger.Log.Debug("Overwriting target config %s", t.Dest)
		err := os.Rename(temp, t.Dest)
		if err != nil {
			if strings.Contains(err.Error(), "device or resource busy") {
				logger.Log.Debug("Rename to %s failed - target is likely a mount. Trying to write instead")
				var contents []byte
				var rerr error
				contents, rerr = ioutil.ReadFile(temp)
				if rerr != nil {
					return rerr
				}
				err := ioutil.WriteFile(t.Dest, contents, t.FileMode)
				if err != nil {
					return err
				}
				os.Chown(t.Dest, t.Uid, t.Gid)

			} else {
				return err
			}
		}
		if t.ReloadCmd != "" {
			if err := t.reload(); err != nil {
				return err
			}
		}
		logger.Log.Info("Target config %s is updated", t.Dest)
	} else {
		logger.Log.Warning("Target config %s in sync", t.Dest)
	}
	return nil
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
