package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hartfordfive/prom-http-sd-server/lib"
	"github.com/hartfordfive/prom-http-sd-server/logger"
	consul "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

type ConsulStore struct {
	client     *consul.Client
	kv         *consul.KV
	allowStale bool
}

var consulKVPrefix string = "prom-http-sd-server"

/*******************************************/

// Lock stores data for a consul lock
type ConsulLock struct {
	path       string
	consulLock *consul.Lock
	c          <-chan struct{}
}

// NewLock creates a new consul lock object for path
func newLock(consulClient *consul.Client, lockPath string) (*ConsulLock, error) {

	l := &ConsulLock{}
	l.path = lockPath

	// if l.consulLock, err = consulClient.LockKey(lockPath); err != nil {
	// 	return nil, err
	// }

	opts := &consul.LockOptions{
		Key: lockPath,
		//Value:      []byte("set by sender 1"),
		SessionTTL: "10s",
		SessionOpts: &consul.SessionEntry{
			//Checks:   []string{"check1", "check2"},
			Behavior: "release",
		},
	}
	lock, err := consulClient.LockOpts(opts)
	if err != nil {
		return nil, err
	}
	l.consulLock = lock

	return l, nil
}

// Lock attempts to lock the consul key
func (l *ConsulLock) lock() (err error) {
	logger.Logger.Debug("Trying to acquire lock",
		zap.String("lock_path", l.path),
	)
	if l.c, err = l.consulLock.Lock(nil); err != nil {
		return err
	}
	logger.Logger.Debug("Lock acquired",
		zap.String("lock_path", l.path),
	)
	return
}

// Unlock releases the consul lock being held
func (l *ConsulLock) unlock() {
	logger.Logger.Debug("Releasing lock",
		zap.String("lock_path", l.path),
	)
	l.consulLock.Unlock()
}

/*****************************************/

func NewConsulDataStore(consulHost string, allowStale bool, shutdownNotify chan bool) *ConsulStore {
	ds := &ConsulStore{
		allowStale: allowStale,
	}

	// Get a new client
	client, err := consul.NewClient(&consul.Config{
		Address: consulHost,
	})
	if err != nil {
		panic(err)
	}
	ds.client = client
	// Get a handle to the KV API
	kv := client.KV()
	ds.kv = kv

	return ds
}

func (s *ConsulStore) getTargetKey(targetGroup, target string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s/targetGroup/%s", consulKVPrefix, targetGroup), "/")
}
func (s *ConsulStore) getLockKey(targetGroup, target string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s-lock/targetGroup/%s", consulKVPrefix, targetGroup), "/")
}

func (s *ConsulStore) AddTargetToGroup(targetGroup, target string) error {

	lKey := s.getLockKey(targetGroup, target)

	logger.Logger.Debug("Getting lock key",
		zap.String("key", lKey),
	)
	l, err := newLock(s.client, lKey)
	if err != nil {
		logger.Logger.Error("Could not create new lock",
			zap.String("key", lKey),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
	}
	if err := l.lock(); err != nil {
		logger.Logger.Error("Could not acquire lock key",
			zap.String("key", lKey),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
	}
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup, target)

	pair, _, err := s.kv.Get(key, &consul.QueryOptions{AllowStale: s.allowStale})
	if err != nil {
		logger.Logger.Error("Could not get target group key",
			zap.String("key", key),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		panic(err)
	}

	tg := &TargetGroup{}

	if pair != nil {

		if err := json.Unmarshal(pair.Value, tg); err != nil {
			logger.Logger.Error("Could not unserialize target group data from consul KV store",
				zap.String("error", err.Error()),
			)
		}

		// Don't add if it's already in the list of targetGroup targets
		if lib.Contains(tg.Targets, target) {
			logger.Logger.Info("Target group already contains target",
				zap.String("target", target),
			)
			return nil
		}

	}

	tg.Targets = append(tg.Targets, target)
	b, err := json.Marshal(tg)

	logger.Logger.Debug("Adding target to consul kv ",
		zap.String("target", target),
		zap.String("key", key),
	)
	p := &consul.KVPair{Key: key, Value: b}
	_, err = s.kv.Put(p, nil)
	if err != nil {
		panic(err)
	}
	return nil
}

func (s *ConsulStore) RemoveTargetFromGroup(targetGroup, target string) error {

	return nil
}

func (s *ConsulStore) AddLabelsToGroup(targetGroup string, labels map[string]string) error {
	// PUT a new KV pair
	p := &consul.KVPair{Key: "REDIS_MAXCLIENTS", Value: []byte("1000")}
	_, err := s.kv.Put(p, nil)
	if err != nil {
		panic(err)
	}
	return nil
}

func (s *ConsulStore) RemoveLabelFromGroup(targetGroup, label string) error {
	return nil
}

func (s *ConsulStore) Serialize(debug bool) (string, error) {

	logger.Logger.Info("Getting keys with prefix ",
		zap.String("prefix", consulKVPrefix),
	)
	keys, _, err := s.kv.Keys(consulKVPrefix, "", &consul.QueryOptions{AllowStale: s.allowStale})
	if err != nil {
		return "", nil
	}

	data := []TargetGroup{}

	for _, k := range keys {
		logger.Logger.Info("Got key",
			zap.String("key", k),
		)
		pair, meta, err := s.kv.Get(k, &consul.QueryOptions{AllowStale: s.allowStale})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Target raw: %+v\n", string(pair.Value))
		fmt.Printf("Target meta: %+v\n", meta)

	}

	res, _ := json.MarshalIndent(data, "", "    ")
	return string(res), nil
}

func (s *ConsulStore) Shutdown() {

}
