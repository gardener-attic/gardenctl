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

package botanist

import (
	"encoding/json"
	"time"

	"github.com/gardener/gardenctl/pkg/client/kubernetes"
	"github.com/gardener/gardenctl/pkg/garden"
	"github.com/gardener/gardenctl/pkg/logger"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// EnsureImagePullSecretsGarden ensures that the image pull secrets do exist in the Garden cluster
// namespace in which the Shoot resource has been created, and that the default service account in
// that namespace contains the respective .imagePullSecrets[] field.
func (b *Botanist) EnsureImagePullSecretsGarden() error {
	return EnsureImagePullSecrets(b.K8sGardenClient, b.Shoot.ObjectMeta.Namespace, b.Secrets, true, b.Logger)
}

// EnsureImagePullSecretsSeed ensures that the image pull secrets do exist in the Seed cluster's
// Shoot namespace and that the default service account in that namespace contains the respective
// .imagePullSecrets[] field.
func (b *Botanist) EnsureImagePullSecretsSeed() error {
	return EnsureImagePullSecrets(b.K8sSeedClient, b.ShootNamespace, b.Secrets, true, b.Logger)
}

// EnsureImagePullSecretsShoot ensures that the image pull secrets do exist in the Shoot cluster's
// kube-system namespace and that the default service account in that namespace contains the
// respective .imagePullSecrets[] field.
func (b *Botanist) EnsureImagePullSecretsShoot() error {
	return EnsureImagePullSecrets(b.K8sShootClient, metav1.NamespaceSystem, b.Secrets, true, b.Logger)
}

// EnsureImagePullSecrets takes a Kubernetes client <k8sClient> and a <namespace> and creates the
// image pull secrets stored in the Garden namespace and having the respective role label. After
// that it patches the default service account in that namespace by appending the names of the just
// created secrets to its .imagePullSecrets[] list.
func EnsureImagePullSecrets(k8sClient kubernetes.Client, namespace string, secrets map[string]*corev1.Secret, createSecrets bool, log *logrus.Entry) error {
	var (
		imagePullKeys       = garden.GetSecretKeysWithPrefix("image-pull", secrets)
		serviceAccountName  = "default"
		serviceAccountPatch = corev1.ServiceAccount{
			ImagePullSecrets: []corev1.LocalObjectReference{},
		}
	)
	if len(imagePullKeys) == 0 {
		return nil
	}

	err := wait.PollImmediate(5*time.Second, 60*time.Second, func() (bool, error) {
		_, err := k8sClient.GetServiceAccount(namespace, serviceAccountName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				msg := `Waiting for ServiceAccount '` + serviceAccountName + `' to be created in namespace '` + namespace + `'...`
				if log != nil {
					log.Info(msg)
				} else {
					logger.Logger.Info(msg)
				}
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	for _, key := range imagePullKeys {
		secret := secrets[key]
		if createSecrets {
			_, err := k8sClient.CreateSecret(namespace, secret.ObjectMeta.Name, corev1.SecretTypeDockercfg, secret.Data, true)
			if err != nil {
				return err
			}
		}
		serviceAccountPatch.ImagePullSecrets = append(serviceAccountPatch.ImagePullSecrets, corev1.LocalObjectReference{
			Name: secret.ObjectMeta.Name,
		})
	}

	patch, err := json.Marshal(serviceAccountPatch)
	if err != nil {
		return err
	}
	_, err = k8sClient.PatchServiceAccount(namespace, serviceAccountName, patch)
	return err
}
