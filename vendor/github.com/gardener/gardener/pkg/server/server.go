// Copyright 2018 The Gardener Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/server/handlers"
)

// Serve starts a HTTP server.
func Serve(k8sGardenClient kubernetes.Client, bindAddress string, port int, metricsInterval time.Duration) {
	http.HandleFunc("/healthz", handlers.Healthz)
	http.Handle("/metrics", handlers.InitMetrics(k8sGardenClient, metricsInterval))

	listenAddress := fmt.Sprintf("%s:%d", bindAddress, port)
	go http.ListenAndServe(listenAddress, nil)
	logger.Logger.Infof("Garden controller manager HTTP server started (serving on %s)", listenAddress)
}
