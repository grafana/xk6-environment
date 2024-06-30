package kubernetes

import (
	"bytes"
	"context"
	"fmt"

	"xk6-environment/pkg/fs"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateTest(ctx context.Context, testName string, td fs.TestDef) error {
	d, err := td.ReadTest()
	if err != nil {
		return err
	}

	if td.IsYaml() {
		// k6-operator mode
		if err := c.Apply(ctx, bytes.NewBufferString(string(d))); err != nil {
			return err
		}

		return nil
	}

	// k6-standalone mode

	// Note: here everything is created in default namespace.
	// If there's a requirement for specific namespace, then probably
	// it should be a k6-operator mode and not a k6-standalone.

	cm := configmapSpec(td.Location, d)
	_, err = c.clientset.CoreV1().ConfigMaps("default").Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	job := jobSpec(testName, td)
	_, err = c.clientset.BatchV1().Jobs("default").Create(ctx, job, metav1.CreateOptions{})
	return err
}

// very minimal definition of job with k6-standalone, to imitate
// the local execution
func jobSpec(testName string, td fs.TestDef) *batchv1.Job {
	var (
		zero32      int32 = 0
		defaultPath       = "/test"
	)
	// this is a copy
	td.Location = defaultPath + "/" + td.Location
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &zero32,
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []apiv1.Container{
						{
							Name:    "k6",
							Image:   fmt.Sprintf("grafana/k6:%s", td.Opts.Version),
							Command: append([]string{"sh", "-c"}, td.Cmd()),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "script-volume",
									MountPath: defaultPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "script-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "script-volume",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

// very minimal definition of script for k6-standalone
func configmapSpec(name string, data []byte) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "script-volume",
		},
		// BinaryData: map[string][]byte{
		// 	name: data,
		// },
		Data: map[string]string{
			name: string(data),
		},
	}

	return cm
}
