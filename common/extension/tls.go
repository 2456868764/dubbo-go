package extension

import (
	"dubbo.apache.org/dubbo-go/v3/tls"
)

var tlsProviders = make(map[string]func() tls.TlsProvider)

// SetTlsProvider sets the TlsProvider extension with @name
func SetTlsProvider(name string, v func() tls.TlsProvider) {
	tlsProviders[name] = v
}

// GetTlsProvider finds the TlsProvider extension with @name
func GetTlsProvider(name string) tls.TlsProvider {
	if protocols[name] == nil {
		panic("tls provider for " + name + " is not existing, make sure you have import the package.")
	}
	return tlsProviders[name]()
}
