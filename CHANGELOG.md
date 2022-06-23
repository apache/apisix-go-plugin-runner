---
title: Changelog
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

## Table of Contents

- [0.4.0](#040)
- [0.3.0](#030)
- [0.2.0](#020)
- [0.1.0](#010)

## 0.4.0

### Feature

- :sunrise: feat: add response-rewrite plugin [#91](https://github.com/apache/apisix-go-plugin-runner/pull/91)
- :sunrise: feat: support response filter for plugin [#90](https://github.com/apache/apisix-go-plugin-runner/pull/90)
- :sunrise: feat: add debugf function [#87](https://github.com/apache/apisix-go-plugin-runner/pull/87)

### Change

- change: add DefaultPlugin so that we don't need to reimplement all the methods [#92](https://github.com/apache/apisix-go-plugin-runner/pull/92)

## 0.3.0

### Feature

- :sunrise: feat: support upstream response header modify [#68](https://github.com/apache/apisix-go-plugin-runner/pull/68)
- :sunrise: feat: support fetch request body [#70](https://github.com/apache/apisix-go-plugin-runner/pull/70)
- :sunrise: feat: introduce context to plugin runner [#63](https://github.com/apache/apisix-go-plugin-runner/pull/63)
- :sunrise: feat: add fault-injection plugin for benchmark [#46](https://github.com/apache/apisix-go-plugin-runner/pull/46)
- :sunrise: feat: add e2e framework [#72](https://github.com/apache/apisix-go-plugin-runner/pull/72)

### Bugfix

- fix: write response header break request [#65](https://github.com/apache/apisix-go-plugin-runner/pull/65)
- fix: addressed blank space of GITSHA populated [#58](https://github.com/apache/apisix-go-plugin-runner/pull/58)
- fix: make sure the cached conf expires after the token [#44](https://github.com/apache/apisix-go-plugin-runner/pull/44)
- fix: avoid reusing nil builder [#42](https://github.com/apache/apisix-go-plugin-runner/pull/42)

## 0.2.0

### Feature

- :sunrise: feat: support Var API [#31](https://github.com/apache/apisix/pull/31)
- :sunrise: feat: provide default APISIX_CONF_EXPIRE_TIME to simplify
  thing [#30](https://github.com/apache/apisix/pull/30)
- :sunrise: feat: handle idempotent key in PrepareConf [#27](https://github.com/apache/apisix/pull/27)

### Bugfix

- fix: a race when reusing flatbuffers.Builder [#35](https://github.com/apache/apisix/pull/35)
- fix: the default socket permission is not enough [#25](https://github.com/apache/apisix/pull/25)

## 0.1.0

### Feature

- First implementation
