/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plugin

import (
	"strconv"
	"sync"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	A6 "github.com/api7/ext-plugin-proto/go/A6"
	pc "github.com/api7/ext-plugin-proto/go/A6/PrepareConf"
	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

var (
	cache *ConfCache
)

type ConfEntry struct {
	Name  string
	Value interface{}
}
type RuleConf []ConfEntry

type ConfCache struct {
	lock sync.Mutex

	tokenCache *ttlcache.Cache
	keyCache   *ttlcache.Cache

	tokenCounter uint32
}

func newConfCache(ttl time.Duration) *ConfCache {
	cc := &ConfCache{
		tokenCounter: 0,
	}
	for _, c := range []**ttlcache.Cache{&cc.tokenCache, &cc.keyCache} {
		cache := ttlcache.NewCache()
		err := cache.SetTTL(ttl)
		if err != nil {
			log.Fatalf("failed to set global ttl for cache: %s", err)
		}
		cache.SkipTTLExtensionOnHit(false)
		*c = cache
	}
	return cc
}

func (cc *ConfCache) Set(req *pc.Req) (uint32, error) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	key := string(req.Key())
	// APISIX < 2.9 doesn't send the idempotent key
	if key != "" {
		res, err := cc.keyCache.Get(key)
		if err == nil {
			return res.(uint32), nil
		}

		if err != ttlcache.ErrNotFound {
			log.Errorf("failed to get cached token with key: %s", err)
			// recreate the token
		}
	}

	entries := RuleConf{}
	te := A6.TextEntry{}
	for i := 0; i < req.ConfLength(); i++ {
		if req.Conf(&te, i) {
			name := string(te.Name())
			plugin := findPlugin(name)
			if plugin == nil {
				log.Warnf("can't find plugin %s, skip", name)
				continue
			}

			log.Infof("prepare conf for plugin %s", name)

			v := te.Value()
			conf, err := plugin.ParseConf(v)
			if err != nil {
				log.Errorf(
					"failed to parse configuration for plugin %s, configuration: %s, err: %v",
					name, string(v), err)
				continue
			}

			entries = append(entries, ConfEntry{
				Name:  name,
				Value: conf,
			})
		}
	}

	cc.tokenCounter++
	token := cc.tokenCounter
	err := cc.tokenCache.Set(strconv.FormatInt(int64(token), 10), entries)
	if err != nil {
		return 0, err
	}

	err = cc.keyCache.Set(key, token)
	return token, err
}

func (cc *ConfCache) SetInTest(token uint32, entries RuleConf) error {
	return cc.tokenCache.Set(strconv.FormatInt(int64(token), 10), entries)
}

func (cc *ConfCache) Get(token uint32) (RuleConf, error) {
	res, err := cc.tokenCache.Get(strconv.FormatInt(int64(token), 10))
	if err != nil {
		return nil, err
	}
	return res.(RuleConf), err
}

func InitConfCache(ttl time.Duration) {
	cache = newConfCache(ttl)
}

func PrepareConf(buf []byte) (*flatbuffers.Builder, error) {
	req := pc.GetRootAsReq(buf, 0)

	token, err := cache.Set(req)
	if err != nil {
		return nil, err
	}

	builder := util.GetBuilder()
	pc.RespStart(builder)
	pc.RespAddConfToken(builder, token)
	root := pc.RespEnd(builder)
	builder.Finish(root)
	return builder, nil
}

func GetRuleConf(token uint32) (RuleConf, error) {
	return cache.Get(token)
}

func SetRuleConfInTest(token uint32, conf RuleConf) error {
	return cache.SetInTest(token, conf)
}
