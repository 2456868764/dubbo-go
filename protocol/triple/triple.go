/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package triple

import (
	"crypto/tls"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/internal"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
	"github.com/dubbogo/gost/log/logger"
)

const (
	// TRIPLE protocol name
	TRIPLE = "tri"
)

var (
	tripleProtocol *TripleProtocol
)

func init() {
	extension.SetProtocol(TRIPLE, GetProtocol)
}

type TripleProtocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex
	serverMap  map[string]*Server
}

// Export TRIPLE service for remote invocation
func (tp *TripleProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	var info *server.ServiceInfo
	infoRaw, ok := url.GetAttribute(constant.ServiceInfoKey)
	if ok {
		info = infoRaw.(*server.ServiceInfo)
	}

	tlsProviderName := url.GetParam(constant.TLSProvider, "")
	var tlsConfigProvider triple_protocol.TLSConfigProvider
	if len(tlsProviderName) > 0 {
		tlsProvider := extension.GetTLSProvider(tlsProviderName)
		tlsConfigProvider = func() (*tls.Config, error) {
			callUrl := url
			callProvider := tlsProvider
			return callProvider.GetServerWorkLoadTLSConfig(callUrl)
		}
	}

	exporter := NewTripleExporter(serviceKey, invoker, tp.ExporterMap())
	tp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[TRIPLE Protocol] Export service: %s", url.String())
	tp.openServer(invoker, info, tlsConfigProvider)
	internal.HealthSetServingStatusServing(url.Service())
	return exporter
}

// *Important*. This function is only for testing. When server package is finished, remove this function
// and modify related tests.
func (tp *TripleProtocol) exportForTest(invoker protocol.Invoker, info *server.ServiceInfo) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	// todo: retrieve this info from url
	exporter := NewTripleExporter(serviceKey, invoker, tp.ExporterMap())
	tp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[TRIPLE Protocol] Export service: %s", url.String())
	tp.openServer(invoker, info, nil)
	internal.HealthSetServingStatusServing(url.Service())
	return exporter
}

func (tp *TripleProtocol) openServer(invoker protocol.Invoker, info *server.ServiceInfo, tlsConfigProvider triple_protocol.TLSConfigProvider) {
	url := invoker.GetURL()
	tp.serverLock.Lock()
	defer tp.serverLock.Unlock()

	if _, ok := tp.serverMap[url.Location]; ok {
		tp.serverMap[url.Location].RefreshService(invoker, info)
		return
	}

	if _, ok := tp.ExporterMap().Load(url.ServiceKey()); !ok {
		panic("[TRIPLE Protocol]" + url.Key() + "is not existing")
	}

	// TODO Set tlsprovider and mutualTLSMode here
	srv := NewServer(tlsConfigProvider)
	srv.Start(invoker, info)

	tp.serverMap[url.Location] = srv
}

// Refer a remote triple service
func (tp *TripleProtocol) Refer(url *common.URL) protocol.Invoker {
	var invoker protocol.Invoker
	var err error

	tlsProviderName := url.GetParam(constant.TLSProvider, "")
	var tlsConfigProvider triple_protocol.TLSConfigProvider
	if len(tlsProviderName) > 0 {
		tlsProvider := extension.GetTLSProvider(tlsProviderName)
		tlsConfigProvider = func() (*tls.Config, error) {
			callUrl := url
			callProvider := tlsProvider
			return callProvider.GetClientWorkLoadTLSConfig(callUrl)
		}
	}

	// for now, we do not need to use this info
	_, ok := url.GetAttribute(constant.ClientInfoKey)
	if ok {
		// stub code generated by new protoc-gen-go-triple
		invoker, err = NewTripleInvoker(url, tlsConfigProvider)
	} else {
		// stub code generated by old protoc-gen-go-triple
		invoker, err = NewDubbo3Invoker(url)
	}
	if err != nil {
		logger.Warnf("can't dial the server: %s", url.Key())
		return nil
	}
	tp.SetInvokers(invoker)
	logger.Infof("[TRIPLE Protocol] Refer service: %s", url.String())
	return invoker
}

func (tp *TripleProtocol) Destroy() {
	logger.Infof("TripleProtocol destroy.")

	tp.serverLock.Lock()
	defer tp.serverLock.Unlock()
	for key, server := range tp.serverMap {
		delete(tp.serverMap, key)
		server.GracefulStop()
	}

	tp.BaseProtocol.Destroy()
}

func NewTripleProtocol() *TripleProtocol {
	return &TripleProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func GetProtocol() protocol.Protocol {
	if tripleProtocol == nil {
		tripleProtocol = NewTripleProtocol()
	}
	return tripleProtocol
}
