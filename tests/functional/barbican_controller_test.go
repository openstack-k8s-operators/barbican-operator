package functional_test

import (
	//"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	//"k8s.io/utils/ptr"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

var _ = Describe("Barbican controller", func() {

	var barbicanName types.NamespacedName
	//var dbSyncJobName types.NamespacedName
	//var bootstrapJobName types.NamespacedName
	//var deploymentName types.NamespacedName

	BeforeEach(func() {

		barbicanName = types.NamespacedName{
			Name:      "barbican",
			Namespace: namespace,
		}
		/*
			dbSyncJobName = types.NamespacedName{
				Name:      "keystone-db-sync",
				Namespace: namespace,
			}
			bootstrapJobName = types.NamespacedName{
				Name:      "keystone-bootstrap",
				Namespace: namespace,
			}
			deploymentName = types.NamespacedName{
				Name:      "keystone",
				Namespace: namespace,
			}
		*/

		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A Barbican instance is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanName, GetDefaultBarbicanSpec()))
		})

		/*
			It("should have the Spec fields defaulted", func() {
				Barbican := GetBarbican(barbicanName)
				Expect(Barbican.Spec.ServiceUser).Should(Equal("barbican"))
				Expect(Barbican.Spec.DatabaseInstance).Should(Equal("openstack"))
				Expect(Barbican.Spec.DatabaseUser).Should(Equal("barbican"))
			})

			It("should have the Status fields initialized", func() {
				Barbican := GetBarbican(barbicanName)
				Expect(Barbican.Status.Hash).To(BeEmpty())
				Expect(Barbican.Status.BarbicanAPIReadyCount).To(Equal(int32(0)))
				Expect(Barbican.Status.BarbicanWorkerReadyCount).To(Equal(int32(0)))
				Expect(Barbican.Status.BarbicanKeystoneListenerReadyCount).To(Equal(int32(0)))
				Expect(Barbican.Status.TransportURLSecret).To(Equal(""))
				Expect(Barbican.Status.DatabaseHostname).To(Equal(""))
			})
		*/

		It("should have input not ready and unknown Conditions initialized", func() {
			/*
				th.ExpectCondition(
					barbicanName,
					ConditionGetterFunc(BarbicanConditionGetter),
					condition.ReadyCondition,
					corev1.ConditionFalse,
				)
			*/

			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
			/*

				for _, cond := range []condition.Type{
					condition.ServiceConfigReadyCondition,
					condition.DBReadyCondition,
					condition.DBSyncReadyCondition,
					//condition.BarbicanAPIReadyCondition,
					//condition.BarbicanWorkerReadyCondition,
					//condition.BarbicanKeystoneListenerReadyCondition,
					//condition.,
					//condition.ExposeServiceReadyCondition,
					//condition.BootstrapReadyCondition,
					//condition.DeploymentReadyCondition,
					condition.NetworkAttachmentsReadyCondition,
					//condition.CronJobReadyCondition,
				} {
					th.ExpectCondition(
						barbicanName,
						ConditionGetterFunc(BarbicanConditionGetter),
						cond,
						corev1.ConditionUnknown,
					)
				}
			*/
		})
	})
})
