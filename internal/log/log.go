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

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.SugaredLogger
)

func newLogger() *zap.SugaredLogger {
	var level = zap.NewAtomicLevel()
	level.SetLevel(zapcore.WarnLevel)

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		os.Stdout,
		level)
	lg := zap.New(core, zap.AddStacktrace(zap.ErrorLevel), zap.AddCaller(), zap.AddCallerSkip(1))
	return lg.Sugar()
}

func init() {
	logger = newLogger()
}

func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}
