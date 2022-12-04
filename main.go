package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v3"
	"szuro.net/crapage/observer"
	"szuro.net/crapage/subject"
	"szuro.net/crapage/zbx"
)

type Target struct {
	Name       string
	Type       string
	Connection string
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

func main() {

	C := ParseCrapageConfig("./crapage.yaml")
	c, _ := zbx.ParseZabbixConfig(C.ServerConfig)

	o := C.Targets[0].ToObserver()
	hs := subject.New()
	hs.Register(o)
	funnel := zbx.FileReaderGenerator(c)
	go hs.AcceptValues(funnel)

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
