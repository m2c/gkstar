package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

var SC *ServerConfig
var Configs *Config
var YamlFile []byte

/**
 * server config
 */
type ServerBaseConfig struct {
	Addr     string `yaml:"addr"`
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"loglevel"`
	Profile  string `yaml:"profile"`
	LogPath  string `yaml:"logPath"`
	LogName  string `yaml:"logName"`
}
type ServerConfig struct {
	SConfigure ServerBaseConfig `yaml:"server"`
}

func init() {
	yamlFile, err := ioutil.ReadFile("application.yaml")
	if err != nil {
		fmt.Println("load application.yaml error, will exit,please fix the application")
	}
	conf := &ServerConfig{}
	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		fmt.Println(err.Error())
		//exit
		os.Exit(0)
	}
	env := conf.SConfigure.Profile
	for _, v := range os.Args {
		arg := strings.Split(v, "=")
		if len(arg) != 2 {
			continue
		}
		if arg[0] == "env" {
			if arg[1] != "dev" && arg[1] != "test" && arg[1] != "prod" {
				panic(fmt.Sprintf("command env %s need dev test prod", arg[1]))
			}
			env = arg[1]
		}
	}
	/*
	* parse the config file
	 */

	if len(env) == 0 {
		// load dev profile application-dev.yaml
		Configs = InitAllConfig("application-dev.yaml")
	} else {
		if strings.EqualFold(env, "dev") {
			Configs = InitAllConfig("application-dev.yaml")
		} else if strings.EqualFold(env, "test") {
			Configs = InitAllConfig("application-test.yaml")
		} else if strings.EqualFold(env, "prod") {
			Configs = InitAllConfig("application-prod.yaml")
		}
	}
	SC = conf
}
