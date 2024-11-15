package config

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"io"
	"os"
	"path"
)

type LdapConf struct {
	Addr     binding.String
	Port     binding.String
	Username binding.String
	Password binding.String
	Limit    binding.Int
}

type LdapConfData struct {
	Addr     string
	Port     string
	Username string
	Password string
	Limit    int
}

func InitLdapCon() *LdapConf {
	res := &LdapConf{
		Addr:     binding.NewString(),
		Port:     binding.NewString(),
		Username: binding.NewString(),
		Password: binding.NewString(),
		Limit:    binding.NewInt(),
	}
	res.load()
	return res
}

func (x *LdapConf) ToData() *LdapConfData {
	res := &LdapConfData{}
	res.Addr, _ = x.Addr.Get()
	res.Port, _ = x.Port.Get()
	res.Username, _ = x.Username.Get()
	res.Password, _ = x.Password.Get()
	res.Limit, _ = x.Limit.Get()
	return res
}

func (x *LdapConf) GetByData(data *LdapConfData) {
	x.Addr.Set(data.Addr)
	x.Port.Set(data.Port)
	x.Username.Set(data.Username)
	x.Password.Set(data.Password)
	x.Limit.Set(data.Limit)
}

func (x *LdapConf) Save() {
	data := x.ToData()
	marshal, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Marshal err:", err)
	}
	file, err := os.Create(getLdapConfigPath())
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return
	}
	defer file.Close()
	file.WriteString(string(marshal))
}

// loadData 用于加载数据
func (x *LdapConf) load() {
	res := new(LdapConfData)
	defer func() {
		x.GetByData(res)
	}()
	file, err := os.Open(getLdapConfigPath())
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("读取文件错误:", err)
		return
	}
	if len(byteValue) > 0 {
		err = json.Unmarshal(byteValue, res)
		if err != nil {
			fmt.Println("Unmarshal err:", err)
			return
		}
	}
}

func getLdapConfigPath() string {
	storageRootURI := fyne.CurrentApp().Storage().RootURI()
	println(storageRootURI.Path())
	return path.Join(storageRootURI.Path(), ldapConfigName)
}
