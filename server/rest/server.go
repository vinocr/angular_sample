//Copyright 2017 Huawei Technologies Co., Ltd
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
package rest

import (
	"crypto/tls"
	"github.com/ServiceComb/service-center/pkg/common"
	"github.com/ServiceComb/service-center/util"
	"github.com/ServiceComb/service-center/util/rest"
	"github.com/astaxie/beego"
	"net/http"
	"time"
)

var (
	defaultRESTfulServer *rest.Server
)

func LoadConfig() (srvCfg *rest.ServerConfig, err error) {
	srvCfg = rest.DefaultServerConfig()
	readHeaderTimeout, _ := time.ParseDuration(beego.AppConfig.DefaultString("read_header_timeout", "60s"))
	readTimeout, _ := time.ParseDuration(beego.AppConfig.DefaultString("read_timeout", "60s"))
	writeTimeout, _ := time.ParseDuration(beego.AppConfig.DefaultString("write_timeout", "60s"))
	maxHeaderBytes := beego.AppConfig.DefaultInt("max_header_bytes", 16384)
	var tlsConfig *tls.Config
	if common.GetServerSSLConfig().SSLEnabled {
		verifyClient := common.GetServerSSLConfig().VerifyClient
		tlsConfig, err = rest.GetServerTLSConfig(verifyClient)
		if err != nil {
			return
		}
	}
	srvCfg.ReadHeaderTimeout = readHeaderTimeout
	srvCfg.ReadTimeout = readTimeout
	srvCfg.WriteTimeout = writeTimeout
	srvCfg.MaxHeaderBytes = maxHeaderBytes
	srvCfg.TLSConfig = tlsConfig
	return
}

func newDefaultServer(addr string, handler http.Handler) error {
	if defaultRESTfulServer != nil {
		return nil
	}
	srvCfg, err := LoadConfig()
	if err != nil {
		return err
	}
	srvCfg.Handler = handler
	defaultRESTfulServer = rest.NewServer(srvCfg)
	util.Logger().Warnf(nil, "listen on server %s.", addr)

	return nil
}

func ListenAndServeTLS(addr string, handler http.Handler) (err error) {
	err = newDefaultServer(addr, handler)
	if err != nil {
		return err
	}
	// 证书已经在config里加载，这里不需要再重新加载
	return defaultRESTfulServer.ListenAndServeTLS("", "")
}
func ListenAndServe(addr string, handler http.Handler) (err error) {
	err = newDefaultServer(addr, handler)
	if err != nil {
		return err
	}
	return defaultRESTfulServer.ListenAndServe()
}

func GracefulStop() {
	if defaultRESTfulServer == nil {
		return
	}
	defaultRESTfulServer.Shutdown()
}

func DefaultServer() *rest.Server {
	return defaultRESTfulServer
}

func NewServer(ep string) (srv *rest.Server, err error) {
	ipAddr, err := util.ParseEndpoint(ep)
	if err != nil {
		return
	}

	srvCfg, err := LoadConfig()
	if err != nil {
		return
	}
	srvCfg.Addr = ipAddr
	srv = rest.NewServer(srvCfg)

	if srvCfg.TLSConfig == nil {
		err = srv.Listen()
	} else {
		err = srv.ListenTLS()
	}
	if err != nil {
		return
	}
	return
}
