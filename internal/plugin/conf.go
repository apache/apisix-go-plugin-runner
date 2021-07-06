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
	"sync/atomic"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	A6 "github.com/api7/ext-plugin-proto/go/A6"
	pc "github.com/api7/ext-plugin-proto/go/A6/PrepareConf"
	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/apache/apisix-go-plugin-runner/internal/util"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

type ConfEntry struct {
	Name  string
	Value interface{}
}
type RuleConf []ConfEntry

var (
	cache        *ttlcache.Cache
	cacheCounter uint32 = 0
)

func InitConfCache(ttl time.Duration) {
	cache = ttlcache.NewCache()
	err := cache.SetTTL(ttl)
	if err != nil {
		log.Fatalf("failed to set global ttl for cache: %s", err)
	}
	cache.SkipTTLExtensionOnHit(false)
	cacheCounter = 0
}

func genCacheToken() uint32 {
	return atomic.AddUint32(&cacheCounter, 1)
}

func PrepareConf(buf []byte) (*flatbuffers.Builder, error) {
	req := pc.GetRootAsReq(buf, 0)
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
					"failed to parse configuration for plugin %s, configuration: %s",
					name, string(v))
				continue
			}

			entries = append(entries, ConfEntry{
				Name:  name,
				Value: conf,
			})
		}
	}

	token := genCacheToken()
	err := cache.Set(strconv.FormatInt(int64(token), 10), entries)
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
	res, err := cache.Get(strconv.FormatInt(int64(token), 10))
	if err != nil {
		return nil, err
	}
	return res.(RuleConf), err
}

func SetRuleConf(token uint32, conf RuleConf) error {
	return cache.Set(strconv.FormatInt(int64(token), 10), conf)
}
