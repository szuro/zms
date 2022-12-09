package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
	"szuro.net/crapage/observer"
	"szuro.net/crapage/subject"
	"szuro.net/crapage/zbx"
)

type Target struct {
	Name       string
	Type       string
	Connection string
	Source     []string
}

func (t *Target) ToObserver() (obs observer.Observer) {
	switch t.Type {
	case "print":
		obs = observer.NewPrint(t.Name, t.Connection)
		// case "azuretable":
	}

	return obs
}

type CrapageConf struct {
	ServerConfig string `yaml:"server_config"`
	Targets      []Target
}

func ParseCrapageConfig(path string) (conf CrapageConf) {
	file, err := os.ReadFile(path)
	if err != nil {
		return
	}

	conf = CrapageConf{}
	err = yaml.Unmarshal(file, &conf)

	return
}

func MkSubjects(zabbix zbx.ZabbixConf) (obs map[string]subject.Subjecter) {
	obs = make(map[string]subject.Subjecter)
	for _, v := range zabbix.ExportTypes {
		switch v {
		case zbx.HISTORY:
			hs := subject.NewSubject[zbx.History]()
			hs.Funnel = zbx.FileReaderGenerator[zbx.History](zabbix)
			obs[zbx.HISTORY] = &hs
		case zbx.TREND:
			ts := subject.NewSubject[zbx.Trend]()
			ts.Funnel = zbx.FileReaderGenerator[zbx.Trend](zabbix)
			obs[zbx.TREND] = &ts
		default:
			fmt.Printf("Not supported export: %s", v)
		}
	}
	return
}

func main() {

	C := ParseCrapageConfig("./crapage.yaml")
	c, _ := zbx.ParseZabbixConfig(C.ServerConfig)

	subjects := MkSubjects(c)

	for _, o := range C.Targets {
		for k, v := range subjects {
			if slices.Contains(o.Source, k) {
				v.Register(o.ToObserver())
			}
		}
	}

	for _, v := range subjects {
		go v.AcceptValues()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		switch <-sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			fmt.Print("Exiting...")
			return
		default:
			return
		}
	}

}
