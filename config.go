package config

import (
	"flag"
	"fmt"
	"github.com/fighterlyt/cfgStore/model"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"reflect"
	"strings"
)

//FileEncoding 配置文件编码类型
type FileEncoding int

const (
	//JSON json编码
	JSON FileEncoding = iota
	//YAML yaml编码
	YAML
)

var (
	UnsupportedError = errors.New("不识别的编码")
)

//Config 配置管理器接口，支持多格式(json,yaml),多配置，优先级 命令行>本地>全局配置
type Config interface {
	/*LoadFile 加载配置文件
	参数:
	*	path     	string      		全局配置文件路径
	*	localPath	string            	本地配置文件路径
	*	encoding 	FileEncoding	配置文件编码
	*	data     	interface{}       	配置文件对象，必须是指针
	返回值:
	*	error	error
	*/
	LoadFile(path, localPath string, encoding FileEncoding, data interface{}) error
	LoadFromServer(url, key string, data interface{}) error

	/*Generate 定义命令行参数
	参数:
	返回值:
	*/
	Generate()
}

//NewConfig 构建一个新的配置文件管理器
func NewConfig() Config {
	return &config{}
}

//config 实现一个配置管理器
type config struct {
	data interface{}
}

/*Generate 生成命令行参数
参数:
返回值:
*/
func (c config) Generate() {
	c.parse(reflect.ValueOf(c.data).Elem(), "")
}

/*parse 解析并生成命令行参数
参数:
*	value 	reflect.Value	配置文件对象
*	prefix	string       	前置，格式为xxx.xxx
返回值:
*/
func (c *config) parse(value reflect.Value, prefix string) {
	fieldCount := value.NumField()
	name := ""
	for i := 0; i < fieldCount; i++ {
		field := value.Field(i)
		if prefix == "" {
			name = strings.ToLower(value.Type().Field(i).Name)
		} else {
			name = strings.ToLower(strings.Join([]string{prefix, value.Type().Field(i).Name}, "."))
		}

		switch field.Type().Kind() {
		case reflect.Struct:
			if prefix == "" {
				c.parse(field, name)
			} else {
				c.parse(field, prefix+"."+name)

			}
		case reflect.String:
			flag.StringVar(field.Addr().Interface().(*string), name, field.String(), name)
		case reflect.Int:
			flag.IntVar(field.Addr().Interface().(*int), name, int(field.Int()), name)
		case reflect.Bool:
			flag.BoolVar(field.Addr().Interface().(*bool), name, field.Bool(), name)
		}
	}
}

/*LoadFile 加载配置文件
参数:
*	path     	string      		全局配置文件路径
*	localPath	string            	本地配置文件路径
*	encoding 	FileEncoding	配置文件编码
*	data     	interface{}       	配置文件对象，必须是指针
返回值:
*	error	error
*/
func (c *config) LoadFile(path, localPath string, encoding FileEncoding, data interface{}) error {
	if err := c.readAndParse(path, encoding, data); err != nil {
		return err
	}
	if localPath != "" {
		if err := c.readAndParse(localPath, encoding, data); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				return err
			}
		}
	}

	c.data = data
	return nil
}

func (c *config) LoadFromServer(url, key string, data interface{}) error {
	if resp, err := http.Get(url + "/" + key); err != nil {
		return errors.Wrap(err, "发起请求")
	} else {
		readResponse(resp)
		defer resp.Body.Close()
		cfg := &model.Config{}
		respData := &model.Response{
			Data: cfg,
		}
		if err = jsoniter.NewDecoder(resp.Body).Decode(respData); err != nil {
			return errors.Wrap(err, "解析数据")
		} else {
			if respData.ErrorCode != 0 {
				return errors.Wrap(err, respData.Error)
			} else {

				reader := strings.NewReader(cfg.Data)
				switch cfg.Type {
				case model.JSON:
					err = c.decode(reader, JSON, data)
				case model.YAML:
					err = c.decode(reader, YAML, data)
				default:
					return UnsupportedError
				}

			}
		}
		if err == nil {
			c.data = data
		}
		return err
	}

}

/*readAndParse 读取文件并解析
参数:
*	path    	string      		文件路径
*	encoding	FileEncoding		文件编码
*	data    	interface{} 		解析目的对象
返回值:
*	error	error
*/
func (c *config) readAndParse(path string, encoding FileEncoding, data interface{}) error {
	if configFile, err := os.Open(path); err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return errors.Wrapf(err, "打开配置文件%s", path)
	} else {
		defer configFile.Close()
		if err = c.decode(configFile, encoding, data); err != nil {
			return errors.Wrapf(err, "解析配置文件%s", path)
		}
	}
	return nil
}
func (c *config) decode(reader io.Reader, encoding FileEncoding, data interface{}) error {
	switch encoding {
	case JSON:
		decoder := jsoniter.NewDecoder(reader)
		if err := decoder.Decode(data); err != nil {
			return errors.Wrap(err, "json解析错误")
		}
	case YAML:
		decoder := yaml.NewDecoder(reader)
		if err := decoder.Decode(data); err != nil {
			return errors.Wrap(err, "yaml解析错误")
		}
	default:
		return UnsupportedError
	}
	return nil
}

func readResponse(resp *http.Response) {
	data, _ := httputil.DumpResponse(resp, true)
	fmt.Println(string(data))
}
