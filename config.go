package conf

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfig(path string, config interface{}) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("read config file %s err %v \n", path, err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(b, config)
	if err != nil {
		fmt.Printf("unmarshal config err: %v\n", err)
		os.Exit(1)
	}
}

func ParseEnvString(envKey string, param ...string) (string, bool) {
	if len(param) > 0 && param[0] != "" {
		return param[0], true
	}
	if v, ok := os.LookupEnv(envKey); ok {
		return v, true
	}
	return "", false
}
