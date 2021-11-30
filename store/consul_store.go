package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hartfordfive/prom-http-sd-server/lib"
	"github.com/hartfordfive/prom-http-sd-server/logger"
	consul "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

type ConsulStore struct {
	client     *consul.Client
	allowStale bool
}

var consulKVPrefix string = "prom-http-sd-server"

/*******************************************/

// Lock stores data for a consul lock
type ConsulLock struct {
	path       string
	consulLock *consul.Lock
	client     *consul.Client
	c          <-chan struct{}
}

func newLock(consulClient *consul.Client, lockPath, lockContents string) (*ConsulLock, error) {

	l := &ConsulLock{
		client: consulClient,
	}
	l.path = lockPath

	opts := &consul.LockOptions{
		Key:        lockPath,
		Value:      []byte(lockContents),
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

func (l *ConsulLock) unlock() {
	logger.Logger.Debug("Releasing lock",
		zap.String("lock_path", l.path),
	)
	l.consulLock.Unlock()
	_, err := l.client.KV().Delete(l.path, nil)
	if err != nil {
		logger.Logger.Error("Could delete lock",
			zap.String("key", l.path),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
	}

}

/*****************************************/

func NewConsulDataStore(consulHost string, allowStale bool, shutdownNotify chan bool) (*ConsulStore, error) {

	host, _, _ := lib.ParseURL(consulHost)
	httpEndpoint := fmt.Sprintf("http://%s/v1/status/leader", host)
	if !lib.CheckHttp2xx(httpEndpoint, 3) {
		return nil, errors.New(fmt.Sprintf("Could not connect to consul at %s", consulHost))
	}

	ds := &ConsulStore{
		allowStale: allowStale,
	}

	// Get a new client
	client, err := consul.NewClient(&consul.Config{
		Address: consulHost,
	})
	if err != nil {
		return nil, err
	}
	ds.client = client
	return ds, nil
}

func (s *ConsulStore) getTargetKey(targetGroup string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s/targetGroup/%s", consulKVPrefix, targetGroup), "/")
}
func (s *ConsulStore) getLockKey(targetGroup string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s-lock/targetGroup/%s", consulKVPrefix, targetGroup), "/")
}

func (s *ConsulStore) getLock(targetGroup, lockContents string) (*ConsulLock, error) {
	lKey := s.getLockKey(targetGroup)

	logger.Logger.Debug("Getting lock key",
		zap.String("key", lKey),
	)
	l, err := newLock(s.client, lKey, lockContents)
	if err != nil {
		logger.Logger.Error("Could not create new lock",
			zap.String("key", lKey),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		return nil, err
	}
	if err := l.lock(); err != nil {
		logger.Logger.Error("Could not acquire lock key",
			zap.String("key", lKey),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		return nil, err
	}
	return l, nil
}

func (s *ConsulStore) AddTargetToGroup(targetGroup, target string) error {

	l, err := s.getLock(targetGroup, fmt.Sprintf("{\"set_at\": \"%s\"}", time.Now().String()))
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup)

	pair, _, err := s.client.KV().Get(key, &consul.QueryOptions{AllowStale: s.allowStale})
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

	if tg.Labels == nil {
		tg.Labels = map[string]string{}
	}
	b, err := json.Marshal(tg)

	logger.Logger.Debug("Adding target to consul kv ",
		zap.String("target", target),
		zap.String("key", key),
	)
	p := &consul.KVPair{Key: key, Value: b}
	_, err = s.client.KV().Put(p, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *ConsulStore) RemoveTargetFromGroup(targetGroup, target string) error {

	l, err := s.getLock(targetGroup, fmt.Sprintf("{\"set_at\": \"%s\"}", time.Now().String()))
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup)

	pair, _, err := s.client.KV().Get(key, &consul.QueryOptions{AllowStale: s.allowStale})
	if err != nil {
		logger.Logger.Error("Could not get target group key",
			zap.String("key", key),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		return err
	}

	tg := &TargetGroup{}

	if pair == nil {
		logger.Logger.Warn("Target group doesn't exist",
			zap.String("target_group", targetGroup),
		)
		return nil
	}

	if err := json.Unmarshal(pair.Value, tg); err != nil {
		logger.Logger.Error("Could not unserialize target group data from consul KV store",
			zap.String("error", err.Error()),
		)
	}

	// If the target isn't in the list, the return immediately, no op to be completed
	if !lib.Contains(tg.Targets, target) {
		return nil
	}

	// Otherwise, update the list by removing the target
	logger.Logger.Debug("Removing target from target group",
		zap.String("target", target),
		zap.String("target_group", targetGroup),
		zap.String("consul_key", key),
	)
	newList := lib.RemoveFromList(tg.Targets, target)
	tg.Targets = newList
	if tg.Labels == nil {
		tg.Labels = map[string]string{}
	}

	b, err := json.Marshal(tg)

	p := &consul.KVPair{Key: key, Value: b}
	if _, err = s.client.KV().Put(p, nil); err != nil {
		return err
	}

	return nil
}

func (s *ConsulStore) RemoveTargetGroup(targetGroup string) error {

	l, err := s.getLock(targetGroup, fmt.Sprintf("{\"set_at\": \"%s\"}", time.Now().String()))
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup)

	_, err = s.client.KV().Delete(key, &consul.WriteOptions{})
	if err != nil {
		logger.Logger.Error("Could note delete target group",
			zap.String("key", key),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		return err
	}

	return nil

}

func (s *ConsulStore) GetTargetGroupLabels(targetGroup string) (*map[string]string, error) {

	l, err := s.getLock(targetGroup, fmt.Sprintf("{\"set_at\": \"%s\"}", time.Now().String()))
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup)

	pair, _, err := s.client.KV().Get(key, &consul.QueryOptions{AllowStale: s.allowStale})
	if err != nil {
		logger.Logger.Error("Could not get target group key",
			zap.String("key", key),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		return nil, err
	}

	tg := &TargetGroup{}

	if pair != nil {
		if err := json.Unmarshal(pair.Value, tg); err != nil {
			logger.Logger.Error("Could not unserialize target group data from consul KV store",
				zap.String("error", err.Error()),
			)
		}
	}

	// b, _ := json.MarshalIndent(tg, "", "    ")
	// res := string(b)
	return &tg.Labels, nil

}

func (s *ConsulStore) AddLabelsToGroup(targetGroup string, labels map[string]string) error {

	l, err := s.getLock(targetGroup, fmt.Sprintf("{\"set_at\": \"%s\"}", time.Now().String()))
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup)

	pair, _, err := s.client.KV().Get(key, &consul.QueryOptions{AllowStale: s.allowStale})
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
	}

	for k, v := range labels {
		tg.Labels[k] = v
	}

	b, err := json.Marshal(tg)

	logger.Logger.Debug("Adding target group labels to consul kv ",
		zap.String("target", targetGroup),
		zap.String("labels", fmt.Sprintf("%v", labels)),
	)
	p := &consul.KVPair{Key: key, Value: b}
	_, err = s.client.KV().Put(p, nil)
	if err != nil {
		panic(err)
	}
	return nil
}

func (s *ConsulStore) RemoveLabelFromGroup(targetGroup, label string) error {

	l, err := s.getLock(targetGroup, fmt.Sprintf("{\"set_at\": \"%s\"}", time.Now().String()))
	defer l.unlock() // if not defered, lock acquision will wait indefinitely

	key := s.getTargetKey(targetGroup)

	pair, _, err := s.client.KV().Get(key, &consul.QueryOptions{AllowStale: s.allowStale})
	if err != nil {
		logger.Logger.Error("Could not get target group key",
			zap.String("key", key),
			zap.String("error", fmt.Sprintf("%s", err.Error())),
		)
		return err
	}

	tg := &TargetGroup{}

	if pair == nil {
		return errors.New(fmt.Sprintf("Label %s doesn't exist in target group %s", label, targetGroup))
	}

	if err := json.Unmarshal(pair.Value, tg); err != nil {
		logger.Logger.Error("Could not unserialize target group data from consul KV store",
			zap.String("error", err.Error()),
		)
	}

	if _, ok := tg.Labels[label]; !ok {
		logger.Logger.Warn("Target group label doesn't exist. Nothign to remove",
			zap.String("target_group", targetGroup),
			zap.String("label", label),
		)
		return nil
	} else {
		delete(tg.Labels, label)
	}

	// Otherwise, update the list by removing the target
	logger.Logger.Debug("Removing label from target group",
		zap.String("target_group", targetGroup),
		zap.String("consul_key", key),
	)

	if tg.Labels == nil {
		tg.Labels = map[string]string{}
	}

	b, err := json.Marshal(tg)

	p := &consul.KVPair{Key: key, Value: b}
	if _, err = s.client.KV().Put(p, nil); err != nil {
		return err
	}

	return nil
}

func (s *ConsulStore) Serialize(debug bool) (string, error) {

	logger.Logger.Info("Getting keys with prefix ",
		zap.String("prefix", consulKVPrefix),
	)
	keys, _, err := s.client.KV().Keys(consulKVPrefix, "", &consul.QueryOptions{AllowStale: s.allowStale})
	if err != nil {
		return "", nil
	}

	if !debug {
		targetGroupList := []TargetGroup{}

		var keyParts []string

		for _, k := range keys {

			logger.Logger.Info("Got key",
				zap.String("key", k),
			)
			pair, _, err := s.client.KV().Get(k, &consul.QueryOptions{AllowStale: s.allowStale})
			if err != nil {
				panic(err)
			}

			if pair != nil {
				keyParts = strings.Split(k, "/")
				tg := &TargetGroup{Name: keyParts[len(keyParts)-1]}
				if err := json.Unmarshal(pair.Value, tg); err != nil {
					logger.Logger.Error("Could not unserialize target group data from consul KV store",
						zap.String("error", err.Error()),
					)
					continue
				}
				targetGroupList = append(targetGroupList, *tg)
			}
		}

		res, _ := json.MarshalIndent(targetGroupList, "", "    ")
		return string(res), nil
	}

	targetGroupList := map[string][]TargetGroup{}

	var keyParts []string

	tgList := []TargetGroup{}

	for _, k := range keys {

		logger.Logger.Info("Got key",
			zap.String("key", k),
		)
		pair, _, err := s.client.KV().Get(k, &consul.QueryOptions{AllowStale: s.allowStale})
		if err != nil {
			panic(err)
		}

		if pair != nil {
			keyParts = strings.Split(k, "/")
			groupName := keyParts[len(keyParts)-1]
			tg := &TargetGroup{Name: groupName}
			if err := json.Unmarshal(pair.Value, tg); err != nil {
				logger.Logger.Error("Could not unserialize target group data from consul KV store",
					zap.String("error", err.Error()),
				)
				continue
			}
			tgList = append(tgList, *tg)
			targetGroupList[groupName] = tgList
		}
	}

	res, _ := json.MarshalIndent(targetGroupList, "", "    ")
	return string(res), nil

}

func (s *ConsulStore) Shutdown() {
	// Method only needs to be present due to interface contstraints.  Nothing to do in this case as the HTTP client doesn't have a shutdown method
}
