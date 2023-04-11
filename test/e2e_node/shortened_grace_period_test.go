/*
Copyright 2023 The Kubernetes Authors.

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

package e2enode

import (
	"context"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	admissionapi "k8s.io/pod-security-admission/api"
	"time"
)

var _ = SIGDescribe("Shortened Grace Period", func() {
	f := framework.NewDefaultFramework("shortened-grace-period")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelBaseline
	ginkgo.Context("When repeatedly deleting pods", func() {
		var podClient *e2epod.PodClient
		ginkgo.BeforeEach(func() {
			podClient = e2epod.NewPodClient(f)
		})
		ginkgo.It("shorter grace period of a second command overrides the longer grace period of a first command", func() {
			const (
				gracePeriod      = 10000
				gracePeriodShort = 10
			)
			ctx := context.Background()
			podName := "test"
			podClient.CreateSync(ctx, getGracePeriodTestPod(podName, gracePeriod))
			err := podClient.Delete(ctx, podName, *metav1.NewDeleteOptions(gracePeriod))
			framework.ExpectNoError(err)
			start := time.Now()
			podClient.DeleteSync(ctx, podName, *metav1.NewDeleteOptions(gracePeriodShort), gracePeriod*time.Second)
			framework.ExpectEqual(time.Since(start) < gracePeriod*time.Second, true, "Failure to shorten grace period")
		})
	})
})

func getGracePeriodTestPod(name string, gracePeriod int64) *v1.Pod {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    name,
					Image:   busyboxImage,
					Command: []string{"sh", "-c"},
					Args: []string{`
term() {
  if [ "$COUNT" -eq 0 ]; then
    echo "SIGINT 1"
  elif [ "$COUNT" -eq 1 ]; then
    echo "SIGINT 2"
    sleep 5
    exit 0
  else
    echo "SIGINT $COUNT"
    exit 1
  fi
  COUNT=$((COUNT + 1))
}
COUNT=0
trap term SIGINT
while true; do
  sleep 1
done
`},
				},
			},
			TerminationGracePeriodSeconds: &gracePeriod,
		},
	}
	return pod
}
