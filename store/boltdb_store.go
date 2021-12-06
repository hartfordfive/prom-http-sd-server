package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/hartfordfive/prom-http-sd-server/logger"
	"go.uber.org/zap"
)

type BoltDBStore struct {
	db *bolt.DB
}

func NewBoltDBDataStore(filePath string, shutdownNotify chan bool) (*BoltDBStore, error) {
	db, err := bolt.Open(filePath, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	s := &BoltDBStore{
		db: db,
	}

	if _, ok := os.LookupEnv("DEBUG"); ok {
		s.collectMetrics(shutdownNotify)
	}
	go func() {
		for {
			select {
			case <-shutdownNotify:
				logger.Logger.Info("Closing database...")
				s.db.Close()
				return
			}

		}
	}()

	return s, err
}

func (s *BoltDBStore) Shutdown() {
	s.db.Close()
}

func (s *BoltDBStore) GetDB() *bolt.DB {
	return s.db
}

func (s *BoltDBStore) AddTargetToGroup(targetGroup, target string) error {
	bucketName := fmt.Sprintf("targets:%s", targetGroup)
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("Could not create bucket for targets: %s", err)
		}

		if err := b.Put([]byte(target), []byte(nil)); err != nil {
			return fmt.Errorf("Could put item into bucket for targets: %s", err)
		}
		return nil
	})
	return err
}

func (s *BoltDBStore) RemoveTargetFromGroup(targetGroup, target string) error {
	bucketName := fmt.Sprintf("targets:%s", targetGroup)
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Delete([]byte(target))
	})
}

func (s *BoltDBStore) RemoveTargetGroup(targetGroup string) error {
	bucketName := fmt.Sprintf("targets:%s", targetGroup)
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.DeleteBucket([]byte(bucketName))
	})
}

func (s *BoltDBStore) AddLabelsToGroup(targetGroup string, labels map[string]string) error {
	bucketName := fmt.Sprintf("labels:%s", targetGroup)
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			logger.Logger.Error("Could not create bukcet")
			return fmt.Errorf("Could not create bucket for target group labels: %s", err)
		}

		for k, v := range labels {
			if err := b.Put([]byte(k), []byte(v)); err != nil {
				logger.Logger.Error(fmt.Sprintf("Could not add label %s=%s : %s", k, v, err))
				return fmt.Errorf("Could put item into bucket for target group labels: %s", err)
			}
			logger.Logger.Debug(fmt.Sprintf("Adding label %s=%s to bucket %s", k, v, bucketName))
		}
		return nil
	})
	return err
}

func (s *BoltDBStore) RemoveLabelFromGroup(targetGroup, label string) error {
	bucketName := fmt.Sprintf("labels:%s", targetGroup)
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Delete([]byte(label))
	})
}

func (s *BoltDBStore) getTargetGroups() ([]string, error) {

	buckets := []string{}

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			if strings.HasPrefix(string(name), "targets:") {
				buckets = append(buckets, string(name))
			}
			return nil
		})
	})
	if err != nil {
		logger.Logger.Error("Could not get target groups", zap.String("message", fmt.Sprintf("%s", err)))
		return nil, err
	}
	return buckets, nil
}

func (s *BoltDBStore) GetTargetGroupLabels(targetGroup string) (*map[string]string, error) {

	labels := map[string]string{}
	err := s.db.View(func(tx *bolt.Tx) error {
		logger.Logger.Debug(fmt.Sprintf("Bucket=%s", targetGroup))
		b := tx.Bucket([]byte(targetGroup))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			labels[string(k)] = string(v)
		}
		return nil
	})

	return &labels, err
}

func (s *BoltDBStore) getTargetLabelsByGroup() (map[string]bool, error) {

	data := map[string]bool{}

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			if strings.HasPrefix(string(name), "labels:") {
				parts := strings.Split(string(name), ":")
				data[parts[1]] = true
			}
			return nil
		})
	})
	if err != nil {
		logger.Logger.Error("Could not get target group labels", zap.String("message", fmt.Sprintf("%s", err)))
		return nil, err
	}
	return data, nil
}

func (s *BoltDBStore) collectMetrics(cancel chan bool) {
	go func(cancel chan bool, s *BoltDBStore) {
		// Get the current state
		prev := s.db.Stats()
		ticker := time.NewTicker(60 * time.Second)

		for {
			select {
			case <-cancel:
				logger.Logger.Debug("Stopping boltDB stats collection")
				return
			case _ = <-ticker.C:
				stats := s.db.Stats()
				diff := stats.Sub(&prev)
				json.NewEncoder(os.Stderr).Encode(diff)
				prev = stats
			}

		}
	}(cancel, s)
}

func (s *BoltDBStore) Serialize(debug bool) (string, error) {
	/*
		[
			{
				"targets": ["10.0.10.2:9100", "10.0.10.3:9100", "10.0.10.4:9100", "10.0.10.5:9100"],
				"labels": {
					"__meta_datacenter": "london",
					"__meta_prometheus_job": "node"
				}
			},
			...
		]
	*/

	targetGroupsBuckets, err := s.getTargetGroups()
	if err != nil {
		logger.Logger.Debug("Could not get target groups")
		return "", err
	}

	if !debug {
		data := []TargetGroup{}

		s.db.View(func(tx *bolt.Tx) error {

			// Itterate over each target group, which are the bucket names, to get the target
			for _, tgName := range targetGroupsBuckets {
				tg := TargetGroup{Name: tgName}
				b := tx.Bucket([]byte(tgName))
				c := b.Cursor()
				targets := []string{}
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					targets = append(targets, string(k))
				}
				tg.Targets = targets
				logger.Logger.Debug(fmt.Sprintf("Getting labels for target group: labels:%s", strings.Split(tgName, ":")[1]))
				labels, _ := s.GetTargetGroupLabels(fmt.Sprintf("labels:%s", strings.Split(tgName, ":")[1]))
				tg.Labels = *labels

				data = append(data, tg)
			}

			return nil

		})

		res, _ := json.MarshalIndent(data, "", "    ")
		return string(res), nil
	}

	// in this case, return a debug view of the data which shows the target group names
	dataDebug := map[string]map[string]interface{}{}
	dataDebug["targets"] = map[string]interface{}{}

	s.db.View(func(tx *bolt.Tx) error {

		// Itterate over each target group, which are the bucket names, to get the target
		for _, tgName := range targetGroupsBuckets {
			tg := TargetGroup{Name: tgName}
			b := tx.Bucket([]byte(tgName))
			c := b.Cursor()
			targets := []string{}
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				targets = append(targets, string(k))
			}
			tg.Targets = targets
			logger.Logger.Debug(fmt.Sprintf("Getting labels for target group: labels:%s", strings.Split(tgName, ":")[1]))
			labels, _ := s.GetTargetGroupLabels(fmt.Sprintf("labels:%s", strings.Split(tgName, ":")[1]))
			tg.Labels = *labels

			dataDebug["targets"][strings.Split(tgName, ":")[1]] = tg
		}
		return nil

	})

	res, _ := json.MarshalIndent(dataDebug, "", "    ")
	return string(res), nil
}
