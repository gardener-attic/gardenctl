// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
package shoot_test

import (
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/controllermanager/controller/shoot"
	"github.com/gardener/gardener/pkg/operation/common"
	. "github.com/onsi/ginkgo/extensions/table"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestShoot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Shoot Suite")
}

var _ = Describe("Shoot Utils", func() {
	Context("Status", func() {
		DescribeTable("#OrWorse",
			func(s1, s2, expected shoot.Status) {
				Expect(s1.OrWorse(s2)).To(Equal(expected))
				Expect(s2.OrWorse(s1)).To(Equal(expected), "not reflexive")
			},
			Entry("healthy - healthy", shoot.StatusHealthy, shoot.StatusHealthy, shoot.StatusHealthy),
			Entry("healthy - progressing", shoot.StatusHealthy, shoot.StatusProgressing, shoot.StatusProgressing),
			Entry("healthy - unhealthy", shoot.StatusHealthy, shoot.StatusUnhealthy, shoot.StatusUnhealthy),
			Entry("progressing - progressing", shoot.StatusProgressing, shoot.StatusProgressing, shoot.StatusProgressing),
			Entry("progressing - unhealthy", shoot.StatusProgressing, shoot.StatusUnhealthy, shoot.StatusUnhealthy),
			Entry("unhealthy - unhealthy", shoot.StatusUnhealthy, shoot.StatusUnhealthy, shoot.StatusUnhealthy),
		)

		DescribeTable("#ConditionStatusToStatus",
			func(status gardenv1beta1.ConditionStatus, expected shoot.Status) {
				Expect(shoot.ConditionStatusToStatus(status)).To(Equal(expected))
			},
			Entry("ConditionTrue", gardenv1beta1.ConditionTrue, shoot.StatusHealthy),
			Entry("ConditionProgressing", gardenv1beta1.ConditionProgressing, shoot.StatusProgressing),
			Entry("ConditionUnknown", gardenv1beta1.ConditionUnknown, shoot.StatusUnhealthy),
			Entry("ConditionFalse", gardenv1beta1.ConditionFalse, shoot.StatusUnhealthy),
		)

		DescribeTable("#ComputeConditionStatus",
			func(conditions []*gardenv1beta1.Condition, expected shoot.Status) {
				Expect(shoot.ComputeConditionStatus(conditions...)).To(Equal(expected))
			},
			Entry("no conditions", nil, shoot.StatusHealthy),
			Entry("true condition", []*gardenv1beta1.Condition{
				{Status: gardenv1beta1.ConditionTrue},
			}, shoot.StatusHealthy),
			Entry("progressing condition", []*gardenv1beta1.Condition{
				{Status: gardenv1beta1.ConditionTrue},
				{Status: gardenv1beta1.ConditionProgressing},
			}, shoot.StatusProgressing),
			Entry("unknown condition", []*gardenv1beta1.Condition{
				{Status: gardenv1beta1.ConditionTrue},
				{Status: gardenv1beta1.ConditionProgressing},
				{Status: gardenv1beta1.ConditionUnknown},
			}, shoot.StatusUnhealthy),
			Entry("false condition", []*gardenv1beta1.Condition{
				{Status: gardenv1beta1.ConditionTrue},
				{Status: gardenv1beta1.ConditionProgressing},
				{Status: gardenv1beta1.ConditionUnknown},
				{Status: gardenv1beta1.ConditionFalse},
			}, shoot.StatusUnhealthy),
		)

		DescribeTable("#BoolToStatus",
			func(b bool, expected shoot.Status) {
				Expect(shoot.BoolToStatus(b)).To(Equal(expected))
			},
			Entry("true", true, shoot.StatusHealthy),
			Entry("false", false, shoot.StatusUnhealthy),
		)

		DescribeTable("#ComputeStatus",
			func(lastOperation *gardenv1beta1.LastOperation, lastError *gardenv1beta1.LastError, conditions []*gardenv1beta1.Condition, expected shoot.Status) {
				Expect(shoot.ComputeStatus(lastOperation, lastError, conditions...)).To(Equal(expected))
			},
			Entry("lastOperation is nil",
				nil, nil, nil, shoot.StatusHealthy),
			Entry("lastOperation.Type is ShootLastOperationTypeCreate",
				&gardenv1beta1.LastOperation{Type: gardenv1beta1.ShootLastOperationTypeCreate}, nil, nil, shoot.StatusHealthy),
			Entry("lastOperation.Type is ShootLastOperationTypeDelete",
				&gardenv1beta1.LastOperation{Type: gardenv1beta1.ShootLastOperationTypeDelete}, nil, nil, shoot.StatusHealthy),
			Entry("lastOperation.Type is ShootLastOperationTypeCreate and lastError is set",
				&gardenv1beta1.LastOperation{Type: gardenv1beta1.ShootLastOperationTypeCreate}, &gardenv1beta1.LastError{}, nil, shoot.StatusUnhealthy),
			Entry("lastOperation.Type is ShootLastOperationTypeDelete and lastError is set",
				&gardenv1beta1.LastOperation{Type: gardenv1beta1.ShootLastOperationTypeDelete}, &gardenv1beta1.LastError{}, nil, shoot.StatusUnhealthy),
			Entry("lastOperation.State is ShootLastOperationStateProcessing with healthy conditions",
				&gardenv1beta1.LastOperation{State: gardenv1beta1.ShootLastOperationStateProcessing}, nil, nil, shoot.StatusHealthy),
			Entry("lastOperation.State is ShootLastOperationStateProcessing with unhealthy conditions",
				&gardenv1beta1.LastOperation{State: gardenv1beta1.ShootLastOperationStateProcessing}, nil, []*gardenv1beta1.Condition{{Status: gardenv1beta1.ConditionFalse}}, shoot.StatusUnhealthy),
			Entry("lastOperation.State is ShootLastOperationStateProcessing with healthy conditions but lastError set",
				&gardenv1beta1.LastOperation{State: gardenv1beta1.ShootLastOperationStateProcessing}, &gardenv1beta1.LastError{}, nil, shoot.StatusUnhealthy),
			Entry("lastOperation.State is neither ShootLastOperationStateProcessing nor ShootLastOperationStateSucceeded with healthy conditions",
				&gardenv1beta1.LastOperation{State: gardenv1beta1.ShootLastOperationStateError}, nil, nil, shoot.StatusUnhealthy),
			Entry("lastOperation.State is ShootLastOperationStateSucceeded with healthy conditions",
				&gardenv1beta1.LastOperation{State: gardenv1beta1.ShootLastOperationStateSucceeded}, nil, nil, shoot.StatusHealthy),
			Entry("lastOperation.State is ShootLastOperationStateSucceeded with unhealthy conditions",
				&gardenv1beta1.LastOperation{State: gardenv1beta1.ShootLastOperationStateSucceeded}, nil, []*gardenv1beta1.Condition{{Status: gardenv1beta1.ConditionFalse}}, shoot.StatusUnhealthy),
		)

		DescribeTable("#StatusLabelTransform",
			func(status shoot.Status, expectedLabels map[string]string) {
				original := &gardenv1beta1.Shoot{}

				modified, err := shoot.StatusLabelTransform(status)(original.DeepCopy())
				Expect(err).NotTo(HaveOccurred())
				modifiedWithoutLabels := modified.DeepCopy()
				modifiedWithoutLabels.Labels = nil
				Expect(modifiedWithoutLabels).To(Equal(original), "not only labels were modified")
				Expect(modified.Labels).To(Equal(expectedLabels))
			},
			Entry("StatusHealthy", shoot.StatusHealthy, map[string]string{
				common.ShootStatus: string(shoot.StatusHealthy),
			}),
			Entry("StatusProgressing", shoot.StatusProgressing, map[string]string{
				common.ShootStatus:    string(shoot.StatusProgressing),
				common.ShootUnhealthy: "true",
			}),
			Entry("StatusUnhealthy", shoot.StatusUnhealthy, map[string]string{
				common.ShootStatus:    string(shoot.StatusUnhealthy),
				common.ShootUnhealthy: "true",
			}),
		)
	})
})
