// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package log

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.SugaredLogger

	loggerInit sync.Once
)

func NewLogger(level zapcore.Level, out zapcore.WriteSyncer) {
	var atomicLevel = zap.NewAtomicLevel()
	atomicLevel.SetLevel(level)

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		out,
		atomicLevel)
	lg := zap.New(core, zap.AddStacktrace(zap.ErrorLevel), zap.AddCaller(), zap.AddCallerSkip(1))
	logger = lg.Sugar()
}

func getLogger() *zap.SugaredLogger {
	loggerInit.Do(func() {
		if logger == nil {
			// logger is not initialized, for example, running `go test`
			NewLogger(zapcore.InfoLevel, os.Stdout)
		}
	})
	return logger
}

func Infof(template string, args ...interface{}) {
	getLogger().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	getLogger().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	getLogger().Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	getLogger().Fatalf(template, args...)
}
