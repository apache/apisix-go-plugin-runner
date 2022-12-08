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

package runner

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/apache/apisix-go-plugin-runner/internal/server"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
)

// RunnerConfig is the configuration of the runner
type RunnerConfig struct {
	// LogLevel is the level of log, default to `zapcore.InfoLevel`
	LogLevel zapcore.Level
	// LogOutput is the output of log, default to `os.Stdout`
	LogOutput zapcore.WriteSyncer
	// Logger will be reused by the framework when it is not nil.
	Logger *zap.SugaredLogger
}

// Run starts the runner and listen the socket configured by environment variable "APISIX_LISTEN_ADDRESS"
func Run(cfg RunnerConfig) {
	if cfg.LogOutput == nil {
		cfg.LogOutput = os.Stdout
	}

	if cfg.Logger == nil {
		log.NewLogger(cfg.LogLevel, cfg.LogOutput)
	} else {
		log.SetLogger(cfg.Logger)
	}

	server.Run()
}
