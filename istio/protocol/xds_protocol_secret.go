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

package protocol

import (
	"time"

	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
)

type SecretProtocol struct {
	secretCache *resources.SecretCache
}

func NewSecretProtocol(secretCache *resources.SecretCache) (*SecretProtocol, error) {
	secretProtocol := &SecretProtocol{
		secretCache: secretCache,
	}
	return secretProtocol, nil
}

func (s *SecretProtocol) ProcessSecret(secret *tls.Secret) error {
	logger.Debugf("[secret protocol] parse envoy tls secret:%s", utils.ConvertJsonString(secret))
	if secret.GetName() == resources.DefaultSecretName {
		if secret.GetTlsCertificate() != nil {
			certificateChain := secret.GetTlsCertificate().GetCertificateChain().GetInlineBytes()
			privateKey := secret.GetTlsCertificate().GetPrivateKey().GetInlineBytes()
			item := &resources.SecretItem{
				CertificateChain: certificateChain,
				PrivateKey:       privateKey,
				CreatedTime:      time.Now(),
				ResourceName:     secret.GetName(),
				// TODO parse ROOTCA and expire time
			}
			s.secretCache.SetWorkload(item)
			certChainPEM, _ := s.secretCache.GetCertificateChainPEM()
			logger.Infof("[secret protocol] Certificate Chain PEM:\n%s", certChainPEM)
			privateKeyPEM, _ := s.secretCache.GetPrivateKeyPEM()
			logger.Infof("[secret protocol] Private Key PEM:\n%s", privateKeyPEM)
		}
	}

	if secret.GetName() == resources.RootCASecretName {
		if secret.GetValidationContext() != nil {
			rootCA := secret.GetValidationContext().GetTrustedCa().GetInlineBytes()
			s.secretCache.SetRoot(rootCA)
			rootCertPEM, _ := s.secretCache.GetRootCertPEM()
			logger.Infof("[secret protocol] Root Cert PEM:\n%s", rootCertPEM)
		}
	}
	return nil
}
