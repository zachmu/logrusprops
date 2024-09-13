// Copyright 2024 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pkglog

import (
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/magiconair/properties"
	"github.com/sirupsen/logrus"
)

func InitFromFile(configFile string) {
	file, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	props, err := properties.Load(file, properties.UTF8)
	if err != nil {
		return
	}

	logrus.SetReportCaller(true)
	rootLogLevel, ok := props.Get("pkglog.rootLogger")
	var level logrus.Level
	if ok {
		level, err = logrus.ParseLevel(rootLogLevel)
		if err != nil {
			panic(err)
		}
	} else {
		level = logrus.InfoLevel
	}

	// We need the lowest level possible or our hook won't be invoked on all log messages
	logrus.SetLevel(logrus.TraceLevel)

	logrus.SetOutput(io.Discard)
	// clear hooks before adding ours, we only want one
	logrus.StandardLogger().Hooks = make(logrus.LevelHooks)
	logrus.AddHook(Hook{props: props, level: level})
}

type Hook struct {
	props *properties.Properties
	level logrus.Level
}

var _ logrus.Hook = (*Hook)(nil)

func (p Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

var formatter = &logrus.JSONFormatter{
	FieldMap: logrus.FieldMap{
		logrus.FieldKeyTime:  "ts",
		logrus.FieldKeyLevel: "level",
		logrus.FieldKeyMsg:   "msg",
		logrus.FieldKeyFile:  "",
	},
	CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
		return frame.Function, ""
	},
}

func (p Hook) Fire(entry *logrus.Entry) error {
	f := entry.Caller.Function
	f = f[:strings.LastIndex(f, ".")]
	loglevel, ok := p.props.Get("pkglog." + f)
	if !ok {
		if entry.Level <= p.level {
			format, _ := formatter.Format(entry)
			os.Stderr.Write(format)
		}
	} else {
		if level, err := logrus.ParseLevel(loglevel); err == nil {
			if entry.Level <= level {
				format, _ := formatter.Format(entry)
				os.Stderr.Write(format)
			}
		}
	}
	return nil
}
