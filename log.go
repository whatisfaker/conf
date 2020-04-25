package conf

import (
	"github.com/whatisfaker/zaptrace/log"
)

const (
	EnvLogLevel = "LOG_LEVEL"
)

type Log struct {
	Level string `yaml:"level,omitempty"` //debug,info,warn,error
	File  string `yaml:"file,omitempty"`
}

func Logger(l ...*Log) *log.Factory {
	lv := parseLogLevel()
	var logConfig *Log
	if len(l) > 0 {
		logConfig = l[0]
	}
	if lv == "" {
		if logConfig != nil && logConfig.Level != "" {
			lv = logConfig.Level
		} else {
			lv = "info"
		}
	}
	if logConfig != nil && logConfig.File != "" {
		return log.NewFileLogger(lv, logConfig.File)
	}
	return log.NewStdLogger(lv)

}

func parseLogLevel() string {
	v, ok := ParseEnvString(EnvLogLevel)
	if ok {
		return v
	}
	return ""
}
