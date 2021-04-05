/*


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

package controllers

import (
	"context"
	"time"

	gonv1 "gonmap/api/v1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GonMapReconciler reconciles a GonMap object
type GonMapReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var reqAfter30s = ctrl.Result{
	Requeue:      true,
	RequeueAfter: time.Second * 30,
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch

func (r *GonMapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("gonmap", req.NamespacedName)
	gm := &gonv1.GonMap{}
	if err := r.Client.Get(context.Background(), req.NamespacedName, gm); err != nil {
		if errors.IsNotFound(err) {
			r.Log.WithValues("gonmap", req.NamespacedName).Info("gonmap not found")
			return ctrl.Result{Requeue: false}, nil
		}
		r.Log.WithValues("gonmap", req.NamespacedName).Error(err, "could not get gonmap")
		return ctrl.Result{}, err
	}

	ns := &corev1.NamespaceList{}
	if err := r.Client.List(context.Background(), ns, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(gm.NamespaceSelector.MatchLabels),
	}); err != nil {
		r.Log.WithValues("gonmap", req.NamespacedName).Error(err, "could not get namespaces")
		return reqAfter30s, err
	}
	for _, n := range ns.Items {
		r.Log.WithValues("ns", n.Name).Info("")
	}
	// need to filter namespace according to a selector
	for _, n := range ns.Items {

		c := &corev1.ConfigMap{}

		c.ObjectMeta = v1.ObjectMeta{
			Namespace: n.Name,
			Name:      gm.ObjectMeta.Name,
		}

		c.Data = gm.Data

		if err := controllerutil.SetControllerReference(gm, c, r.Scheme); err != nil {
			r.Log.WithValues("gonmap", req.NamespacedName, "namespace", n.ObjectMeta.Name).Error(err, "could not set controller reference")
			return ctrl.Result{}, err
		}

		if err := r.Client.Create(context.Background(), c); err != nil {
			if errors.IsAlreadyExists(err) {
				if err := r.Client.Update(context.Background(), c); err != nil {
					r.Log.WithValues("gonmap", req.NamespacedName, "namespace", n.ObjectMeta.Name, "configmap", gm.Name).Error(err, "error updating existing configmap")
				}
				r.Log.WithValues("gonmap", req.NamespacedName, "namespace", n.ObjectMeta.Name, "configmap", gm.Name).Info("updating because configmap already exists")
				continue
			}
			r.Log.WithValues("gonmap", req.NamespacedName, "namespace", n.ObjectMeta.Name).Error(err, "could not create child configmap")
		}
	}
	return reqAfter30s, nil
}

func (r *GonMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gonv1.GonMap{}).
		Complete(r)
}
