package datasource

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"strings"
	"time"
)

const (
	// DomainPrefix domains of accesslist prefix
	DomainPrefix = "/domains/"
	// SubnetsPrefix subnets of accesslist prefix
	SubnetsPrefix = "/subnets/"
	authInfoLen   = 4
)

type watchCallback func(string) error

type client struct {
	*clientv3.Client
	watches map[string]clientv3.WatchChan
}

func newEtcdclient(machines []string, cert, key, caCert string, basicAuth bool, username string, password string) (*client, error) {
	var cli *clientv3.Client
	watches := make(map[string]clientv3.WatchChan)

	cfg := clientv3.Config{
		Endpoints:   machines,
		DialTimeout: 5 * time.Second,
	}

	if basicAuth {
		cfg.Username = username
		cfg.Password = password
	}

	tlsEnabled := false
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	if caCert != "" {
		certBytes, err := ioutil.ReadFile(caCert)
		if err != nil {
			return &client{cli, watches}, err
		}

		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(certBytes)

		if ok {
			tlsConfig.RootCAs = caCertPool
		}
		tlsEnabled = true
	}

	if cert != "" && key != "" {
		tlsCert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return &client{cli, watches}, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
		tlsEnabled = true
	}

	if tlsEnabled {
		cfg.TLS = tlsConfig
	}

	cli, err := clientv3.New(cfg)
	if err != nil {
		return &client{cli, watches}, err
	}
	return &client{cli, watches}, nil
}

func (c *client) getValues(key string) (map[string]string, error) {
	vars := make(map[string]string)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	defer cancel()
	resp, err := c.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		return vars, err
	}
	for _, ev := range resp.Kvs {
		vars[string(ev.Key)] = string(ev.Value)
	}
	return vars, nil
}

func (c *client) getAccessListConfig() (*accessListConfig, error) {
	alc := new(accessListConfig)
	domains := []string{}
	domainMap, err := c.getValues(DomainPrefix)
	if err != nil {
		return nil, err
	}

	for k := range domainMap {
		domains = append(domains, strings.TrimPrefix(k, DomainPrefix))
	}
	subnets := []string{}
	subnetsMap, err := c.getValues(SubnetsPrefix)
	if err != nil {
		return nil, err
	}
	for k := range subnetsMap {
		subnets = append(subnets, strings.TrimPrefix(k, SubnetsPrefix))
	}

	alc.Domains = domains
	alc.Subnets = subnets
	return alc, nil
}

// AccessListFromEtcd get urlactions from etcd
func AccessListFromEtcd(path string) (*AccessList, error) {
	re := regexp.MustCompile("(.+):(.+)@(.+)")
	params := re.FindStringSubmatch(path)
	if len(params) != authInfoLen {
		return nil, fmt.Errorf("path format error:%s", path)
	}
	c, err := newEtcdclient([]string{params[3]}, "", "", "", true, params[1], params[2])
	if err != nil {
		return nil, err
	}
	alc, err := c.getAccessListConfig()
	if err != nil {
		return nil, err
	}
	al, err := newAccessList(alc)
	if err != nil {
		return nil, err
	}

	logrus.Debug("get acl from etcd: %+v", al)

	ctx := context.Background()

	go func() {
		ctxnew, cancel := context.WithCancel(ctx)
		defer cancel()
		c.watchUpdate(ctxnew, DomainPrefix, al.AddDomain, al.DeleteDomain)
	}()
	go func() {
		ctxnew, cancel := context.WithCancel(ctx)
		defer cancel()
		c.watchUpdate(ctxnew, SubnetsPrefix, al.AddSubnet, al.DeleteSubnet)
	}()
	return al, err
}

func (c *client) watchUpdate(ctx context.Context, prefix string, add, del watchCallback) {
	var err error
	rch := c.Watch(ctx, prefix, clientv3.WithPrefix())
	for {
		wresp := <-rch
		for _, ev := range wresp.Events {
			k := strings.TrimPrefix(string(ev.Kv.Key), prefix)
			switch ev.Type {
			case clientv3.EventTypePut:
				err = add(k)
			case clientv3.EventTypeDelete:
				err = del(k)
			}
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
					"event": ev,
				}).Error("watch update failed")
			} else {
				logrus.WithFields(logrus.Fields{
					"event": ev,
					"key":   k,
				}).Debug("watch update")
			}
		}
	}
}

// WatchPrefix watch etcd for keys prefixed by prefix.
func (c *client) WatchPrefix(ctx context.Context, prefix string) error {
	//var err error
	//rch := c.watches[prefix]
	//if rch == nil {
	//c.watches[prefix] = rch
	//}

	//for {
	//select {
	//case <-ctx.Done():
	//return ctx.Err()
	//case wresp := <-rch:
	//for _, ev := range wresp.Events {
	//// Only return if we have a key prefix we care about.
	//// This is not an exact match on the key so there is a chance
	//// we will still pickup on false positives. The net win here
	//// is reducing the scope of keys that can trigger updates.
	//for _, k := range keys {
	//if strings.HasPrefix(string(ev.Kv.Key), k) {
	//return err
	//}
	//}
	//}
	//}
	//}
	//return 0, err
	return nil
}
