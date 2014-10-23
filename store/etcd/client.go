package etcd

import (
	"errors"
	"strings"
	"sync"
	"time"

	goetcd "github.com/coreos/go-etcd/etcd"
)

type Client struct {
	Client *goetcd.Client
}

/*
*	New Etcd cluster connections return *etcd.Client
*	If Connection to cluster can not made,  return error
 */
func NewClient(machines []string, cert, key, caCert string) (*Client, error) {
	var c *goetcd.Client
	var err error
	if key != "" && cert != "" {
		c, err = goetcd.NewTLSClient(machines, cert, key, caCert)
		if err != nil {
			return &Client{c}, err
		}
	} else {
		c = goetcd.NewClient(machines)
	}
	c.SetDialTimeout(time.Duration(3) * time.Second)
	success := c.SetCluster(machines)
	if !success {
		err = errors.New("can not connect to etcd cluster: " + strings.Join(machines, ", "))
	}
	return &Client{c}, err
}

//implement Store.Client interface, GetValues method
func (c *Client) GetValues(keys []string) (map[string]string, error) {
	var waitGroup sync.WaitGroup
	values := make(map[string]string)
	for _, key := range keys {
		waitGroup.Add(1)
		go fetchValue(c, key, values, &waitGroup)
	}
	waitGroup.Wait()
	return values, nil
}

func fetchValue(c *Client, key string, values map[string]string, waitGroup *sync.WaitGroup) error {
	defer waitGroup.Done()
	resp, err := c.Client.Get(key, true, true)
	if err != nil {
		return err
	}
	err = nodesErgodic(resp.Node, values)
	return err
}

func nodesErgodic(node *goetcd.Node, values map[string]string) error {
	if node != nil {
		key := node.Key
		if !node.Dir {
			values[key] = node.Value
		} else {
			for _, subNode := range node.Nodes {
				nodesErgodic(subNode, values)
			}
		}
	}
	return nil
}

func (c *Client) WatchPrefix(prefix string, waitIndex uint64, stopChan chan bool) (uint64, error) {
	if waitIndex == 0 {
		resp, err := c.Client.Get(prefix, false, true)
		if err != nil {
			return 0, err
		}
		return resp.EtcdIndex, nil
	}
	resp, err := c.Client.Watch(prefix, waitIndex+1, true, nil, stopChan)
	return resp.EtcdIndex, err
}
