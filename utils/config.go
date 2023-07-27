package utils

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	//配置文件解析结果
	configRet gjson.Result
	// appPath is the absolute path to the app
	appPath string
	// appConfigPath is the path to the config files
	appConfigPath string
}

func NewConfig(dir string, name string) *Config {
	config := &Config{}
	var err error
	if config.appPath, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
		panic(err)
	}
	workPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	//如果工作目录下不存在，从可执行文件目录读取
	config.appConfigPath = filepath.Join(workPath, dir, name)
	if _, err := os.Stat(config.appConfigPath); err != nil {
		config.appConfigPath = filepath.Join(config.appPath, dir, name)
		if _, err := os.Stat(config.appConfigPath); err != nil {
			panic("file " + dir + "/" + name + " not exist")
		}
	}
	//打开文件
	file, err := os.Open(config.appConfigPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	//读取文件内容
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	config.configRet = gjson.Parse(string(fileContent))
	return config
}

func (config *Config) DecodeSubConfig(configName string, targetConfig interface{}) {
	configRet := config.configRet.Get(configName)
	if !configRet.Exists() {
		panic("conf doesn't include "+configName)
	}
	err := json.Unmarshal([]byte(configRet.Raw), targetConfig)
	if err != nil {
		panic(err)
	}
}

func (config *Config) DecodeConfig(targetConfig interface{}) {
	err := json.Unmarshal([]byte(config.configRet.Raw), targetConfig)
	if err != nil {
		panic(err)
	}
}

/**
	预定义配置
 */

type MySqlDb struct {
	Ip string
	ReadOnlyIp string
	Port int
	Username string
	Password string
	Dbname string
	Charset string
}

func (db *MySqlDb) GetDsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		db.Username,
		db.Password,
		db.Ip,
		db.Port,
		db.Dbname,
		db.Charset)
}

func (db *MySqlDb) GetReadOnlyDsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		db.Username,
		db.Password,
		db.ReadOnlyIp,
		db.Port,
		db.Dbname,
		db.Charset)
}

type MongoDb struct {
	Ip string
}

type HBase struct {
	Type string
	Host string
	Port int
	AccessKey string
	AccessSecret string
}

type Redis struct{
	Ip string
	Port int
	Password string
}