package e2e

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	coreapi "k8s.io/api/core/v1"

	ntoconfig "github.com/openshift/cluster-node-tuning-operator/pkg/config"
	util "github.com/openshift/cluster-node-tuning-operator/test/e2e/util"
)

// Test the application (and rollback) of a custom profile via node labelling.
var _ = ginkgo.Describe("[basic][custom_node_labels] Node Tuning Operator custom profile, node labels", func() {
	const (
		profileHugepages   = "../../../examples/hugepages.yaml"
		nodeLabelHugepages = "tuned.openshift.io/hugepages"
		sysctlVar          = "vm.nr_hugepages"
	)

	ginkgo.Context("custom profile: node labels", func() {
		var (
			node *coreapi.Node
		)

		// Cleanup code to roll back cluster changes done by this test even if it fails in the middle of ginkgo.It()
		ginkgo.AfterEach(func() {
			ginkgo.By("cluster changes rollback")
			if node != nil {
				util.ExecAndLogCommand("oc", "label", "node", "--overwrite", node.Name, nodeLabelHugepages+"-")
			}
			util.ExecAndLogCommand("oc", "delete", "-n", ntoconfig.OperatorNamespace(), "-f", profileHugepages)
		})

		ginkgo.It(fmt.Sprintf("%s set", sysctlVar), func() {
			ginkgo.By("getting a list of worker nodes")
			nodes, err := util.GetNodesByRole(cs, "worker")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(len(nodes)).NotTo(gomega.BeZero(), "number of worker nodes is 0")

			node = &nodes[0]
			ginkgo.By(fmt.Sprintf("getting a tuned pod running on node %s", node.Name))
			pod, err := util.GetTunedForNode(cs, node)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By(fmt.Sprintf("getting the current value of %s in pod %s", sysctlVar, pod.Name))
			valOrig, err := util.GetSysctl(sysctlVar, pod)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By(fmt.Sprintf("labelling node %s with label %s", node.Name, nodeLabelHugepages))
			_, _, err = util.ExecAndLogCommand("oc", "label", "node", "--overwrite", node.Name, nodeLabelHugepages+"=")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By(fmt.Sprintf("creating the custom hugepages profile %s", profileHugepages))
			_, _, err = util.ExecAndLogCommand("oc", "create", "-n", ntoconfig.OperatorNamespace(), "-f", profileHugepages)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("ensuring the custom worker node profile was set")
			err = util.EnsureSysctl(pod, sysctlVar, "16")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By(fmt.Sprintf("deleting the custom hugepages profile %s", profileHugepages))
			_, _, err = util.ExecAndLogCommand("oc", "delete", "-n", ntoconfig.OperatorNamespace(), "-f", profileHugepages)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By(fmt.Sprintf("ensuring the original %s value (%s) is set in pod %s", sysctlVar, valOrig, pod.Name))
			err = util.EnsureSysctl(pod, sysctlVar, valOrig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By(fmt.Sprintf("removing label %s from node %s", nodeLabelHugepages, node.Name))
			_, _, err = util.ExecAndLogCommand("oc", "label", "node", "--overwrite", node.Name, nodeLabelHugepages+"-")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
