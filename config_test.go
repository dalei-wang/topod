package main

import (
	"reflect"
	"testing"
)

func TestInitDefaultConfig(t *testing.T) {
	expect := Config{
		Store:      "etcd",
		StoreNodes: []string{"127.0.0.1:4001"},
		ConfDir:    "/etc/topod/conf.d/",
		Schema:     "http",
		Watch:      false,
		Interval:   60,
		Prefix:     "/",
		Daemon:     false,
		Verbose:    false,
		Noop:       false,
		Debug:      false,
	}
	if err := initConfig(); err != nil {
		t.Errorf(err.Error())
	}
	if !reflect.DeepEqual(config, expect) {
		t.Errorf("Init default config = %v, expect, %v", config, expect)
	}
}
