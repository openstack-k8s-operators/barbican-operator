/*
Copyright 2026.

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

package v1beta1

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestBarbicanSpecDefaultWithUnifiedImage(t *testing.T) {
	g := NewWithT(t)

	_ = os.Setenv("RELATED_IMAGE_BARBICAN_IMAGE_URL_DEFAULT", "barbican-unified-image")
	SetupDefaults()
	defer func() {
		_ = os.Unsetenv("RELATED_IMAGE_BARBICAN_IMAGE_URL_DEFAULT")
		SetupDefaults()
	}()

	spec := &BarbicanSpec{}
	spec.Default()

	g.Expect(spec.BarbicanAPI.ContainerImage).To(Equal("barbican-unified-image"))
	g.Expect(spec.BarbicanWorker.ContainerImage).To(Equal("barbican-unified-image"))
	g.Expect(spec.BarbicanKeystoneListener.ContainerImage).To(Equal("barbican-unified-image"))
}

func TestBarbicanSpecDefaultWithNoEnvVar(t *testing.T) {
	g := NewWithT(t)

	_ = os.Unsetenv("RELATED_IMAGE_BARBICAN_IMAGE_URL_DEFAULT")
	SetupDefaults()

	spec := &BarbicanSpec{}
	spec.Default()

	g.Expect(spec.BarbicanAPI.ContainerImage).To(Equal(BarbicanContainerImage))
	g.Expect(spec.BarbicanWorker.ContainerImage).To(Equal(BarbicanContainerImage))
	g.Expect(spec.BarbicanKeystoneListener.ContainerImage).To(Equal(BarbicanContainerImage))
}

func TestBarbicanSpecDefaultPreservesExplicitImages(t *testing.T) {
	g := NewWithT(t)

	_ = os.Setenv("RELATED_IMAGE_BARBICAN_IMAGE_URL_DEFAULT", "barbican-unified-image")
	SetupDefaults()
	defer func() {
		_ = os.Unsetenv("RELATED_IMAGE_BARBICAN_IMAGE_URL_DEFAULT")
		SetupDefaults()
	}()

	spec := &BarbicanSpec{}
	spec.BarbicanAPI.ContainerImage = "api-override"
	spec.BarbicanWorker.ContainerImage = "worker-override"
	spec.BarbicanKeystoneListener.ContainerImage = "listener-override"
	spec.Default()

	g.Expect(spec.BarbicanAPI.ContainerImage).To(Equal("api-override"))
	g.Expect(spec.BarbicanWorker.ContainerImage).To(Equal("worker-override"))
	g.Expect(spec.BarbicanKeystoneListener.ContainerImage).To(Equal("listener-override"))
}
