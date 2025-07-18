package controller

import (
	"context"
	"fmt"
	"os"

	upgradeapi "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2edeployment "k8s.io/kubernetes/test/e2e/framework/deployment"
	"k8s.io/utils/pointer"
)

type DeploymentOption func(*appsv1.Deployment)

func NewDeployment(name string, opt ...DeploymentOption) *appsv1.Deployment {
	labels := map[string]string{
		upgradeapi.LabelController:    name,
		"app.kubernetes.io/name":      name,
		"app.kubernetes.io/component": "controller",
	}
	securityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer.Bool(false),
		RunAsNonRoot:             pointer.Bool(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsUser:  pointer.Int64(65534),
		RunAsGroup: pointer.Int64(65534),
	}
	deployment := e2edeployment.NewDeployment(name, 1, labels, "system-upgrade-controller", "rancher/system-upgrade-controller:latest", appsv1.RecreateDeploymentStrategyType)
	deployment.Spec.Template.Spec.Volumes = []corev1.Volume{{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}},
	}
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		container.SecurityContext = securityContext
		container.Env = []corev1.EnvVar{{
			Name:  "SYSTEM_UPGRADE_CONTROLLER_NAME",
			Value: name,
		}, {
			Name:  "SYSTEM_UPGRADE_CONTROLLER_LEADER_ELECT",
			Value: "true",
		}, {
			Name:  "SYSTEM_UPGRADE_CONTROLLER_DEBUG",
			Value: "true",
		}, {
			Name: "SYSTEM_UPGRADE_CONTROLLER_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		}, {
			Name: "SYSTEM_UPGRADE_CONTROLLER_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		}}
		container.VolumeMounts = []corev1.VolumeMount{{
			Name:      `tmp`,
			MountPath: "/tmp",
		}}
	}
	for _, fn := range opt {
		fn(deployment)
	}
	return deployment
}

func DeploymentWithTolerations(toleration ...corev1.Toleration) DeploymentOption {
	return func(deployment *appsv1.Deployment) {
		deployment.Spec.Template.Spec.Tolerations = append(deployment.Spec.Template.Spec.Tolerations, toleration...)
	}
}

func DeploymentDefaultTolerations() DeploymentOption {
	return DeploymentWithTolerations()
}

func DeploymentWithServiceAccountName(serviceAcountName string) DeploymentOption {
	return func(deployment *appsv1.Deployment) {
		deployment.Spec.Template.Spec.ServiceAccountName = serviceAcountName
	}
}

func DeploymentImage(image string) DeploymentOption {
	return func(deployment *appsv1.Deployment) {
		deployment.Spec.Template.Spec.Containers[0].Image = image
	}
}

func DeploymentDefaultImage() DeploymentOption {
	if img, ok := os.LookupEnv("SYSTEM_UPGRADE_CONTROLLER_IMAGE"); ok {
		return DeploymentImage(img)
	}
	return DeploymentImage("rancher/system-upgrade-controller:latest")
}

func CreateDeployment(client clientset.Interface, namespace string, deploymentObj *appsv1.Deployment) (*appsv1.Deployment, error) {
	deploymentRes, err := client.AppsV1().Deployments(namespace).Create(context.TODO(), deploymentObj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("deployment %q Create API error: %v", deploymentObj.Name, err)
	}
	e2eframework.Logf("Waiting deployment %q to complete", deploymentObj.Name)
	err = e2edeployment.WaitForDeploymentComplete(client, deploymentRes)
	if err != nil {
		return nil, fmt.Errorf("deployment %q failed to complete: %v", deploymentObj.Name, err)
	}
	return deploymentRes, nil
}
