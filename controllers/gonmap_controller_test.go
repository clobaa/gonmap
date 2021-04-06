package controllers

import (
	"context"
	"fmt"
	gonv1 "gonmap/api/v1"
	"time"

	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	// ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	apiVersion      = "mondo.github.io.clobaa/v1"
	kind            = "GonMap"
	GonMapName      = "gonmap-test"
	GonMapNamespace = "default"
	timeout         = time.Second * 30
	interval        = time.Second * 1
	Ns1             = "ns1"
	Ns2             = "ns2"
	Ns3             = "ns3"
)

var (
	gmLookupKey       = types.NamespacedName{Name: GonMapName, Namespace: GonMapNamespace}
	LabelsForSelector = map[string]string{
		"foo": "bar",
	}
	gmDataLabels = map[string]string{
		// sample data
		"ENV":     "staging",
		"VERSION": "v1.3",
	}

	sampleNs = []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   Ns1,
				Labels: LabelsForSelector,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   Ns2,
				Labels: LabelsForSelector,
			},
		},
	}

	newNs = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   Ns3,
			Labels: LabelsForSelector,
		},
	}

	newNsList = append(sampleNs, newNs)
)

// TODO: Abstract out common code
var _ = Describe("GonMap controller", func() {

	Context("Applying GonMap yaml", func() {
		It("Should create a new GonMap resource", func() {
			createYm()
		})

		It("Should Create ConfigMaps in all the namespaces", func() {
			time.Sleep(time.Second * 10)
			cmList := &corev1.ConfigMapList{}
			nsList := &corev1.NamespaceList{}
			err := k8sClient.List(context.Background(), cmList, &client.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "failed to fetch all the configmaps")
			err = k8sClient.List(context.Background(), nsList)
			Expect(err).NotTo(HaveOccurred(), "failed to fetch all the namespaces")

			By("Expecting to find CMs in all the namespaces")
			Eventually(func() error {
				nsList := &corev1.NamespaceList{}
				err := k8sClient.List(context.Background(), nsList)
				if err != nil {
					// log.Error(err, "failed to fetch all the namespaces")
					return err

				}

				for _, ns := range nsList.Items {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return nil

			}, interval, timeout).Should(Succeed())

		})

		It("Should Delete all the ConfigMaps in all the namespaces on deletion", func() {

			// TODO: Convert this into debug log (V:4)
			// log := ctrl.Log.WithName("controllers").WithName("GonMap")

			By("Expecting to delete YM successfully")
			Eventually(func() error {
				gm := &gonv1.GonMap{}
				k8sClient.Get(context.Background(), gmLookupKey, gm)
				// log.WithValues("gm.GetName()", gm.GetName(), "gm.GetNamespace()", gm.GetNamespace()).Info("")

				return k8sClient.Delete(context.Background(), gm, &client.DeleteOptions{})
			}, timeout, interval).Should(Succeed())

			By("Expecting YM delete to finish")
			Eventually(func() error {
				gm := &gonv1.GonMap{}
				return k8sClient.Get(context.Background(), gmLookupKey, gm)
			}, timeout, interval).ShouldNot(Succeed())

			By("Expecting to delete children CMs to delete successfully")
			Eventually(func() error {
				nsList := &corev1.NamespaceList{}
				err := k8sClient.List(context.Background(), nsList)
				if err != nil {
					// log.Error(err, "failed to fetch all the namespaces")
					return nil

				}

				for _, ns := range nsList.Items {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return errors.New("")

			}, timeout, interval).ShouldNot(Succeed())

		})
	})

	Context("Applying GonMap yaml with Namespace Selector", func() {
		It("Should create a new GonMap resource with Namespace Selector", func() {
			ctx := context.Background()
			createYm()

			By("Expecting to create namespaces successfully")
			for _, ns := range sampleNs {
				Expect(k8sClient.Create(ctx, &ns)).NotTo(HaveOccurred(), fmt.Sprintf("failed to create ns %v", ns.GetName()))
				Eventually(func() error {
					nsCreated := &corev1.Namespace{}
					nsKey := types.NamespacedName{
						Name: ns.GetName(),
					}
					return k8sClient.Get(ctx, nsKey, nsCreated)
				}, timeout, interval).Should(Succeed())
			}

		})

		It("Should create configmaps in all the namespaces matching the ns selector", func() {
			By("Expecting to find YM children CMs matching the namespace")
			Eventually(func() error {

				for _, ns := range sampleNs {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return nil

			}, timeout, interval).Should(Succeed())

			By("Expecting CMs not to be present in namespaces which don't match the selector")
			Eventually(func() error {
				allNs := &corev1.NamespaceList{}
				err := k8sClient.List(context.Background(), allNs)
				if err != nil {
					// log.Error(err, "failed to fetch all the namespaces")
					return nil

				}

				for _, ns := range allNs.Items {
					if containsNs(sampleNs, &ns) {
						continue
					}
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return errors.New("")

			}, timeout, interval).ShouldNot(Succeed())
		})

		It("Should Delete all the CMs in all the namespaces matching the selector on deletion", func() {

			// log := ctrl.Log.WithName("controllers").WithName("GonMap")

			By("Expecting to delete YM successfully")
			Eventually(func() error {
				gm := &gonv1.GonMap{}
				k8sClient.Get(context.Background(), gmLookupKey, gm)
				// log.WithValues("gm.GetName()", gm.GetName(), "gm.GetNamespace()", gm.GetNamespace()).Info("")

				return k8sClient.Delete(context.Background(), gm, &client.DeleteOptions{})
			}, timeout, interval).Should(Succeed())

			By("Expecting YM delete to finish")
			Eventually(func() error {
				gm := &gonv1.GonMap{}
				return k8sClient.Get(context.Background(), gmLookupKey, gm)
			}, timeout, interval).ShouldNot(Succeed())

			By("Expecting children CMs to delete successfully")
			Eventually(func() error {
				for _, ns := range sampleNs {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return errors.New("")

			}, timeout, interval).ShouldNot(Succeed())

			By("Expecting sample namespaces to delete successfully")
			for _, ns := range sampleNs {
				err := k8sClient.Delete(context.Background(), &ns, &client.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				nsLookupKey := types.NamespacedName{Name: ns.GetName()}

				Eventually(func() error {
					deletedNs := corev1.Namespace{}
					err := k8sClient.Get(context.Background(), nsLookupKey, &deletedNs)
					if err != nil {
						// log.Error(err, "cm error")
						// TODO: Check for not found error instead
						// instead of returning success for any error
						return err
					}

					return nil

				}, timeout, interval).ShouldNot(Succeed())
			}

		})
	})
	Context("Applying GonMap yaml with Namespace Selector and adding a new namespace matching the selector after creation of YM", func() {
		It("Should create a new GonMap resource with Namespace Selector", func() {
			ctx := context.Background()
			createYm()

			By("Expecting to create namespaces successfully")
			for _, ns := range sampleNs {
				Expect(k8sClient.Create(ctx, &ns)).NotTo(HaveOccurred(), fmt.Sprintf("failed to create ns %v", ns.GetName()))
				Eventually(func() error {
					nsCreated := &corev1.Namespace{}
					nsKey := types.NamespacedName{
						Name: ns.GetName(),
					}
					return k8sClient.Get(ctx, nsKey, nsCreated)
				}, timeout, interval).Should(Succeed())
			}

		})

		It("Should create configmaps in all the namespaces matching the ns selector", func() {
			By("Expecting to find YM children CMs matching the namespace")
			Eventually(func() error {

				for _, ns := range sampleNs {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return nil

			}, timeout, interval).Should(Succeed())

			By("Expecting CMs not to be present in namespaces which don't match the selector")
			Eventually(func() error {
				allNs := &corev1.NamespaceList{}
				err := k8sClient.List(context.Background(), allNs)
				if err != nil {
					// log.Error(err, "failed to fetch all the namespaces")
					return nil

				}

				for _, ns := range allNs.Items {
					if containsNs(sampleNs, &ns) {
						continue
					}
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return errors.New("")

			}, timeout, interval).ShouldNot(Succeed())
		})

		It("Should should inject CM in newly creates ns which matches the selector", func() {
			By("Expecting to create namespaces successfully")

			ctx := context.Background()
			Expect(k8sClient.Create(ctx, &newNs)).NotTo(HaveOccurred(), fmt.Sprintf("failed to create ns %v", newNs.GetName()))
			Eventually(func() error {
				nsCreated := &corev1.Namespace{}
				nsKey := types.NamespacedName{
					Name: newNs.GetName(),
				}
				return k8sClient.Get(ctx, nsKey, nsCreated)
			}, timeout, interval).Should(Succeed())

			By("Expecting to find YM children CMs matching the namespace")
			Eventually(func() error {

				for _, ns := range newNsList {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return nil

			}, timeout, interval).Should(Succeed())

			By("Expecting CMs not to be present in namespaces which don't match the selector")
			Eventually(func() error {
				allNs := &corev1.NamespaceList{}
				err := k8sClient.List(context.Background(), allNs)
				if err != nil {
					// log.Error(err, "failed to fetch all the namespaces")
					return nil

				}

				for _, ns := range allNs.Items {
					if containsNs(newNsList, &ns) {
						continue
					}
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return errors.New("")

			}, timeout, interval).ShouldNot(Succeed())

		})

		It("Should Delete all the CMs in all the namespaces matching the selector on deletion", func() {

			// log := ctrl.Log.WithName("controllers").WithName("GonMap")

			By("Expecting to delete YM successfully")
			Eventually(func() error {
				gm := &gonv1.GonMap{}
				k8sClient.Get(context.Background(), gmLookupKey, gm)
				// log.WithValues("gm.GetName()", gm.GetName(), "gm.GetNamespace()", gm.GetNamespace()).Info("")

				return k8sClient.Delete(context.Background(), gm, &client.DeleteOptions{})
			}, timeout, interval).Should(Succeed())

			By("Expecting YM delete to finish")
			Eventually(func() error {
				gm := &gonv1.GonMap{}
				return k8sClient.Get(context.Background(), gmLookupKey, gm)
			}, timeout, interval).ShouldNot(Succeed())

			By("Expecting to children CMs to delete successfully")
			Eventually(func() error {
				for _, ns := range newNsList {
					cm := &corev1.ConfigMap{}
					cmLookupKey := types.NamespacedName{Name: GonMapName, Namespace: ns.GetName()}
					err := k8sClient.Get(context.Background(), cmLookupKey, cm)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}
				}

				return errors.New("")

			}, timeout, interval).ShouldNot(Succeed())

			By("Expecting sample namespaces to delete successfully")
			for _, ns := range newNsList {
				err := k8sClient.Delete(context.Background(), &ns, &client.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() error {
					nsLookupKey := types.NamespacedName{Name: ns.GetName()}
					err := k8sClient.Get(context.Background(), nsLookupKey, &ns)
					if err != nil {
						// log.Error(err, "cm error")
						return err
					}

					return nil

				}, timeout, interval).ShouldNot(Succeed())
			}

		})
	})

})

func containsNs(nsList []corev1.Namespace, ns *corev1.Namespace) bool {
	for _, n := range nsList {
		if ns.GetName() == n.GetName() {
			return true
		}
	}

	return false
}

func createYm() {
	ctx := context.Background()
	gm := &gonv1.GonMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GonMapName,
			Namespace: GonMapNamespace,
		},
		Data: gmDataLabels,
		NamespaceSelector: metav1.LabelSelector{
			// Match all namespaces
			MatchLabels: map[string]string{},
		},
	}
	err := k8sClient.Create(ctx, gm)
	Expect(err).NotTo(HaveOccurred(), "failed to create gonmap")

	Eventually(
		func() bool {
			gmCreated := &gonv1.GonMap{}
			err := k8sClient.Get(ctx, gmLookupKey, gmCreated)
			if err != nil {
				return false
			}
			return gmCreated.GetName() == GonMapName
		}, timeout, interval,
	).Should(BeTrue())
}
