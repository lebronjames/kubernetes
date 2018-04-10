/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"net/http"
	goruntime "runtime"

	"github.com/prometheus/client_golang/prometheus"

	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/util/configz"
)

// BuildHandlerChain builds a handler chain with a base handler and CompletedConfig.
func BuildHandlerChain(apiHandler http.Handler, c *CompletedConfig) http.Handler {
	requestContextMapper := apirequest.NewRequestContextMapper()
	requestInfoResolver := &apirequest.RequestInfoFactory{}
	failedHandler := genericapifilters.Unauthorized(requestContextMapper, legacyscheme.Codecs, false)

	handler := genericapifilters.WithAuthorization(apiHandler, requestContextMapper, c.Authorization.Authorizer, legacyscheme.Codecs)
	handler = genericapifilters.WithAuthentication(handler, requestContextMapper, c.Authentication.Authenticator, failedHandler)
	handler = genericapifilters.WithRequestInfo(handler, requestInfoResolver, requestContextMapper)
	handler = apirequest.WithRequestContext(handler, requestContextMapper)
	handler = genericfilters.WithPanicRecovery(handler)

	return handler
}

// NewBaseHandler takes in CompletedConfig and returns a handler.
func NewBaseHandler(c *CompletedConfig) http.Handler {
	mux := mux.NewPathRecorderMux("controller-manager")
	healthz.InstallHandler(mux)
	if c.ComponentConfig.EnableProfiling {
		routes.Profiling{}.Install(mux)
		if c.ComponentConfig.EnableContentionProfiling {
			goruntime.SetBlockProfileRate(1)
		}
	}
	configz.InstallHandler(mux)
	mux.Handle("/metrics", prometheus.Handler())

	return mux
}