package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dynamicgo/gomesh"

	config "github.com/dynamicgo/go-config"
	extend "github.com/dynamicgo/go-config-extend"
	"github.com/dynamicgo/go-config/source"
	"github.com/dynamicgo/go-config/source/envvar"
	"github.com/dynamicgo/go-config/source/file"
	flagsource "github.com/dynamicgo/go-config/source/flag"
	"github.com/dynamicgo/slf4go"
)

var logger slf4go.Logger

// Run run gomesh app
func Run(appname string) {
	logger = slf4go.Get(appname)

	defer func() {
		logger.InfoF("mesh app exit")
		time.Sleep(time.Second * 2)
	}()

	logger.InfoF("start app %s", appname)

	configpath := flag.String("config", "", "special the mesh app config file")

	flag.Parse()

	config, err := createConfig(*configpath)

	if err != nil {
		logger.ErrorF("create config from %s error %s", *configpath, err)
		return
	}

	logconfig, err := extend.SubConfig(config, "slf4go")

	if err != nil {
		logger.Info(fmt.Sprintf("get slf4go config error: %s", err))
		return
	}

	if err := slf4go.Load(logconfig); err != nil {
		logger.Info(fmt.Sprintf("load slf4go config error: %s", err))
		return
	}

	if err := gomesh.Start(config); err != nil {
		logger.Info("start gomesh error: %s", err)
		return
	}

	var c chan int

	logger.InfoF("start app %s -- success", appname)

	<-c
}

func createConfig(configpath string) (config.Config, error) {
	configs, err := loadconfigs(configpath)

	if err != nil {
		return nil, err
	}

	config := config.NewConfig()

	sources := []source.Source{
		envvar.NewSource(envvar.WithPrefix()),
		flagsource.NewSource(),
	}

	for _, path := range configs {
		sources = append(sources, file.NewSource(file.WithPath(path)))
	}

	err = config.Load(sources...)

	return config, err
}

func loadconfigs(path string) ([]string, error) {
	fi, err := os.Stat(path)

	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return []string{
			path,
		}, nil
	}

	var files []string

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if path == "." || path == ".." {
			return err
		}

		files = append(files, path)
		return err
	})

	if err != nil {
		return nil, err
	}

	return files, err
}
