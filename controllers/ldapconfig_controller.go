/*
Copyright 2023.

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
	"fmt"

	//"io"
	"os"
	"strconv"

	//"strings"
	"time"

	//"golang.org/x/crypto/ssh/terminal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"webhook-operator/api/v1alpha1"
	ldapv1alpha1 "webhook-operator/api/v1alpha1"
)

// LdapconfigReconciler reconciles a Ldapconfig object
type LdapconfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ldap.tokenservice.com,resources=ldapconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ldap.tokenservice.com,resources=ldapconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ldap.tokenservice.com,resources=ldapconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=create;get
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=create;get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ldapconfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *LdapconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("reconcile called")
	ldapinst := &ldapv1alpha1.Ldapconfig{}
	err := r.Get(ctx, req.NamespacedName, ldapinst)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			lg.Info("ldapconfig not found so creating one")
			cfg := &ldapv1alpha1.Ldapconfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "webhook-operator-system"},
				Spec:       ldapv1alpha1.LdapconfigSpec{}}
			r.Create(ctx, cfg)
			return ctrl.Result{}, nil
		} else {
			lg.Error(err, "An error occured while getting ldap config ")
			return ctrl.Result{}, err
		}
	}
	cfg := &ldapv1alpha1.Ldapconfig{}
	err = r.Get(ctx, req.NamespacedName, cfg)
	if (v1alpha1.LdapconfigSpec{}) == cfg.Spec {
		lg.Info("Found empty ldap config, nothing to do")
		return ctrl.Result{}, nil
	}
	desired := getDesired(cfg)
	current := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: "ldap", Namespace: "webhook-operator-system"}, current)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		lg.Info("Creating a new Deployment", "Deployment.Namespace", desired.Namespace, "Deployment.Name", desired.Name)
		er := r.Create(ctx, desired)
		if er != nil {
			lg.Info("volume mount is name  " + desired.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)
			lg.Error(er, "Error creating deployment")
			return ctrl.Result{}, er
		}
		lg.Info("Waiting for the deployment to be ready")
		cfg.Status.Condition = "Waiting for the deployment to be ready"
		cfg.Status.Ready = false
		r.Update(ctx, cfg)
		return ctrl.Result{RequeueAfter: time.Duration(30 * float64(time.Second))}, nil
	}
	if current.Status.AvailableReplicas == 0 {
		lg.Info("Available replicas 0. Waiting for the deployment to be ready")
		return ctrl.Result{RequeueAfter: time.Duration(30 * float64(time.Second))}, nil
	}
	/*
		if !reflect.DeepEqual(*current, *desired) {
			lg.Info("Found difference in deployment spec, reconciling")
			r.Update(ctx, desired)
		}*/
	lg.Info("Found difference in deployment spec, reconciling")
	r.Update(ctx, desired)
	restcfg := &rest.Config{}
	_, err = os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if os.IsNotExist(err) {
		kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)
		restconfig, er := kubeconfig.ClientConfig()
		if er != nil {
			lg.Error(er, "Error when creating restconfig from kubeconfig")
		}
		restcfg = restconfig

	} else {
		restconfig, er := rest.InClusterConfig()
		if er != nil {
			lg.Error(er, "Error while getting restconfig from incluster config")
		}
		restcfg = restconfig
	}

	coreclient, err := corev1client.NewForConfig(restcfg)
	if err != nil {
		lg.Error(err, "Error when creating coreclient from restconfig")
	}

	namespace := "webhook-operator-system"

	pods, err := coreclient.Pods(namespace).List(ctx, v1.ListOptions{LabelSelector: "app=ldap"})
	if err != nil {
		lg.Error(err, "Error while getting pod")
	}
	pod := pods.Items[0]

	cfg.Status.Condition = "Setting up diretory server"
	cfg.Status.Ready = false
	r.Update(ctx, cfg)
	anon := make(map[bool]string)
	anon[true] = "on"
	anon[false] = "off"
	cmdlist := []string{"out=$(dsconf localhost backend suffix list|grep -i " + cfg.Spec.Config.Backendname +
		");if test -z \"$out\";then dsconf localhost backend create --suffix " + cfg.Spec.Config.Basedn + " --be-name " + cfg.Spec.Config.Backendname +
		"; else echo \"backend already exists\"; fi",
		"out=$(cat /data/config/container.inf | grep -i \"^basedn\"); if test -z \"$out\";then echo \"basedn = " +
			cfg.Spec.Config.Basedn + "\" >> /data/config/container.inf && dsidm localhost init; else echo \"basedn entry already exists\"; fi",
		"dsconf localhost config replace nsslapd-allow-anonymous-access=" + anon[cfg.Spec.Config.Allowanonymous],
		"dsconf localhost config replace nsslapd-require-secure-binds=" + anon[cfg.Spec.Config.Tls.RequireSSlBinds],
	}
	if cfg.Spec.Config.Tls.KeyCert != "" {
		cmdlist = append(cmdlist, "dsctl localhost tls import-server-key-cert /data/tls/server.crt /data/tls/server.key")
	}
	if cfg.Spec.Config.ExtendedConfig != "" {
		cm := &corev1.ConfigMap{}
		r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: cfg.Spec.Config.ExtendedConfig}, cm)
		schema := cm.Data["99user.ldif"]
		cmdlist = append(cmdlist, "echo \""+schema+"\" > /etc/dirsrv/slapd-localhost/schema/99user.ldif")
	}
	cmd := []string{"/bin/sh", "-c"}
	cmds := ""
	for i, e := range cmdlist {
		cmds = cmds + e
		if i < len(cmdlist)-1 {
			cmds = cmds + " && "
		}
	}
	cmd = append(cmd, cmds)
	fmt.Println("cmd generated", cmd)

	rq := coreclient.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   cmd,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restcfg, "POST", rq.URL())
	if err != nil {
		lg.Error(err, "Error during execute command")
	}
	// Connect this process' std{in,out,err} to the remote shell process.
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		lg.Error(err, "Error during stream output")
	}

	cfg.Status.Condition = "Application ready"
	cfg.Status.Ready = true
	r.Update(ctx, cfg)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LdapconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ldapv1alpha1.Ldapconfig{}).Owns(&appsv1.Deployment{}).
		Complete(r)
}

func getDesired(cfg *v1alpha1.Ldapconfig) *appsv1.Deployment {
	labelselector := make(map[string]string)
	labelselector["app"] = "ldap"
	vmounts, vols := getVolumes(cfg)
	ports := []corev1.ContainerPort{
		{
			Name:          "ldap",
			ContainerPort: 3389,
		},
		{
			Name:          "ldaps",
			ContainerPort: 3636,
		},
	}
	conts := []corev1.Container{
		corev1.Container{
			Name:            "ldap",
			Image:           "pando85/389ds",
			Ports:           ports,
			VolumeMounts:    vmounts,
			ImagePullPolicy: corev1.PullAlways,
			Env: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "DS_ERRORLOG_LEVEL",
					Value: cfg.Spec.Config.Loglevel,
				},
				corev1.EnvVar{
					Name:  "DS_DM_PASSWORD",
					Value: cfg.Spec.Config.Password,
				},
				corev1.EnvVar{
					Name:  "DS_REINDEX",
					Value: strconv.FormatBool(cfg.Spec.Config.Reindex),
				},
			},
		},
	}

	depl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "webhook-operator-system"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labelselector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: "389-ds", Labels: labelselector},
				Spec: corev1.PodSpec{
					Volumes:    vols,
					Containers: conts,
				},
			},
		},
	}
	return depl
}

func getVolumes(cfg *ldapv1alpha1.Ldapconfig) ([]corev1.VolumeMount, []corev1.Volume) {
	vols := []corev1.Volume{}
	vmnts := []corev1.VolumeMount{}
	vols = append(vols, corev1.Volume{
		Name: "hostvolume",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: cfg.Spec.Config.Stg.HostPath,
			},
		},
	})
	vmnts = append(vmnts, corev1.VolumeMount{
		Name:      "hostvolume",
		MountPath: "/data",
	})

	if cfg.Spec.Config.Tls.KeyCert != "" {
		vols = append(vols, corev1.Volume{
			Name: "keyandcert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cfg.Spec.Config.Tls.KeyCert,
				},
			},
		})
		vmnts = append(vmnts, corev1.VolumeMount{
			Name:      "keyandcert",
			MountPath: "/data/tls",
		})
	}
	if cfg.Spec.Config.Tls.Ca != "" {
		vols = append(vols, corev1.Volume{
			Name: "ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cfg.Spec.Config.Tls.Ca,
				},
			},
		})
		vmnts = append(vmnts, corev1.VolumeMount{
			Name:      "ca",
			MountPath: "/data/tls/ca",
		})
	}
	return vmnts, vols

}
