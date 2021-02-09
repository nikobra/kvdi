/*
Copyright 2020,2021 Avi Zimmerman

This file is part of kvdi.

kvdi is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

kvdi is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with kvdi.  If not, see <https://www.gnu.org/licenses/>.
*/

package v1alpha1

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/tinyzimmer/kvdi/pkg/apis/meta/v1"
	"github.com/tinyzimmer/kvdi/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TODO (Do something about this)
var desktopUserID = 9000

// GetStaticEnvVars returns the environment variables configured in the template.
func (t *DesktopTemplate) GetStaticEnvVars() []corev1.EnvVar { return t.Spec.Env }

// GetEnvTemplates returns the environment variable templates.
func (t *DesktopTemplate) GetEnvTemplates() map[string]string { return t.Spec.EnvTemplates }

// GetPulseServer returns the pulse server to give to the proxy for handling audio streams.
func (t *DesktopTemplate) GetPulseServer() string {
	if t.Spec.Config != nil && t.Spec.Config.PulseServer != "" {
		return strings.TrimPrefix(t.Spec.Config.PulseServer, "unix://")
	}
	return fmt.Sprintf("/run/user/%d/pulse/native", desktopUserID)
}

// GetVolumes returns the additional volumes to apply to a pod.
func (t *DesktopTemplate) GetVolumes() []corev1.Volume {
	if t.Spec.VolumeConfig != nil && t.Spec.VolumeConfig.Volumes != nil {
		return t.Spec.VolumeConfig.Volumes
	}
	return nil
}

// GetVolumeMounts returns the additional volume mounts to apply to the desktop container.
func (t *DesktopTemplate) GetVolumeMounts() []corev1.VolumeMount {
	if t.Spec.VolumeConfig != nil && t.Spec.VolumeConfig.VolumeMounts != nil {
		return t.Spec.VolumeConfig.VolumeMounts
	}
	return nil
}

// GetVolumeDevices returns the additional volume devices to apply to the desktop container.
func (t *DesktopTemplate) GetVolumeDevices() []corev1.VolumeDevice {
	if t.Spec.VolumeConfig != nil && t.Spec.VolumeConfig.VolumeDevices != nil {
		return t.Spec.VolumeConfig.VolumeDevices
	}
	return nil
}

// GetInitSystem returns the init system used by the docker image in this template.
func (t *DesktopTemplate) GetInitSystem() DesktopInit {
	if t.Spec.Config != nil && t.Spec.Config.Init != "" {
		return t.Spec.Config.Init
	}
	return InitSystemd
}

// RootEnabled returns true if desktops booted from the template should allow
// users to use sudo.
func (t *DesktopTemplate) RootEnabled() bool {
	if t.Spec.Config != nil {
		return t.Spec.Config.AllowRoot
	}
	return false
}

// FileTransferEnabled returns true if desktops booted from the template should
// allow file transfer.
func (t *DesktopTemplate) FileTransferEnabled() bool {
	if t.Spec.Config != nil {
		return t.Spec.Config.AllowFileTransfer
	}
	return false
}

// GetKVDIVNCProxyImage returns the kvdi-proxy image for the desktop instance.
func (t *DesktopTemplate) GetKVDIVNCProxyImage() string {
	if t.Spec.Config != nil && t.Spec.Config.ProxyImage != "" {
		return t.Spec.Config.ProxyImage
	}
	return fmt.Sprintf("ghcr.io/tinyzimmer/kvdi:kvdi-proxy-%s", version.Version)
}

// GetDesktopImage returns the docker image to use for instances booted from
// this template.
func (t *DesktopTemplate) GetDesktopImage() string {
	return t.Spec.Image
}

// GetDesktopPullPolicy returns the image pull policy for this template.
func (t *DesktopTemplate) GetDesktopPullPolicy() corev1.PullPolicy {
	if t.Spec.ImagePullPolicy != "" {
		return t.Spec.ImagePullPolicy
	}
	return corev1.PullIfNotPresent
}

// GetDesktopPullSecrets returns the pull secrets for this instance.
func (t *DesktopTemplate) GetDesktopPullSecrets() []corev1.LocalObjectReference {
	return t.Spec.ImagePullSecrets
}

// GetDesktopResources returns the resource requirements for this instance.
func (t *DesktopTemplate) GetDesktopResources() corev1.ResourceRequirements {
	return t.Spec.Resources
}

// IsTCPDisplaySocket returns true if the VNC server is listening on a TCP socket.
func (t *DesktopTemplate) IsTCPDisplaySocket() bool {
	return strings.HasPrefix(t.GetDisplaySocketURI(), "tcp://")
}

// IsUNIXDisplaySocket returns true if the VNC server is listening on a UNIX socket.
func (t *DesktopTemplate) IsUNIXDisplaySocket() bool {
	return strings.HasPrefix(t.GetDisplaySocketURI(), "unix://")
}

// GetDisplaySocketAddress returns just the address portion of the display socket URI.
func (t *DesktopTemplate) GetDisplaySocketAddress() string {
	return strings.TrimPrefix(strings.TrimPrefix(t.GetDisplaySocketURI(), "unix://"), "tcp://")
}

// GetDisplaySocketURI returns the display socket URI to pass to the nonvnc-proxy.
func (t *DesktopTemplate) GetDisplaySocketURI() string {
	if t.Spec.Config != nil && t.Spec.Config.SocketAddr != "" {
		return t.Spec.Config.SocketAddr
	}
	return v1.DefaultDisplaySocketAddr
}

// GetDesktopEnvVars returns the environment variables for a desktop pod.
func (t *DesktopTemplate) GetDesktopEnvVars(desktop *Desktop) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  v1.UserEnvVar,
			Value: desktop.GetUser(),
		},
		{
			Name:  "UID",
			Value: strconv.Itoa(desktopUserID), // TODO: Better here than in the images, but still needs refactoring
		},
	}
	if t.IsUNIXDisplaySocket() {
		envVars = append(envVars, corev1.EnvVar{
			Name:  v1.VNCSockEnvVar,
			Value: t.GetDisplaySocketAddress(),
		})
	}
	if t.RootEnabled() {
		envVars = append(envVars, corev1.EnvVar{
			Name:  v1.EnableRootEnvVar,
			Value: "true",
		})
	}
	if static := t.GetStaticEnvVars(); static != nil {
		envVars = append(envVars, static...)
	}
	return envVars
}

// GetDesktopPodSecurityContext returns the security context for pods booted
// from this template.
func (t *DesktopTemplate) GetDesktopPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot: &v1.FalseVal,
	}
}

// GetDesktopContainerSecurityContext returns the container security context for
// pods booted from this template.
func (t *DesktopTemplate) GetDesktopContainerSecurityContext() *corev1.SecurityContext {
	capabilities := make([]corev1.Capability, 0)
	var privileged bool
	var user int64
	if t.GetInitSystem() == InitSystemd {
		// The method of using systemd-logind to trigger a systemd --user process
		// requires CAP_SYS_ADMIN. Specifically, SECCOMP spawning. There might
		// be other ways around this by just using system unit files for everything.
		capabilities = append(capabilities, "SYS_ADMIN")
		privileged = true
		user = 0
	} else {
		privileged = false
		user = int64(desktopUserID)
	}
	if t.Spec.Config != nil {
		capabilities = append(capabilities, t.Spec.Config.Capabilities...)
	}
	return &corev1.SecurityContext{
		Privileged: &privileged,
		RunAsUser:  &user,
		Capabilities: &corev1.Capabilities{
			Add: capabilities,
		},
	}
}

var (
	tmpVolume       = "tmp"
	runVolume       = "run"
	shmVolume       = "shm"
	tlsVolume       = "tls"
	homeVolume      = "home"
	cgroupsVolume   = "cgroups"
	runLockVolume   = "run-lock"
	vncSockVolume   = "vnc-sock"
	pulseSockVolume = "pulse-sock"
)

func (t *DesktopTemplate) needsDedicatedPulseVolume() bool {
	if t.IsUNIXDisplaySocket() {
		if filepath.Dir(t.GetDisplaySocketAddress()) == filepath.Dir(t.GetPulseServer()) {
			return false
		}
	}
	for _, path := range []string{v1.DesktopTmpPath, v1.DesktopRunPath, "/home"} {
		if strings.HasPrefix(t.GetPulseServer(), path) {
			return false
		}
	}
	return true
}

// GetDesktopVolumes returns the volumes to mount to desktop pods.
func (t *DesktopTemplate) GetDesktopVolumes(cluster *VDICluster, desktop *Desktop) []corev1.Volume {
	// Common volumes all containers will need.
	volumes := []corev1.Volume{
		corev1.Volume{
			Name: tmpVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: runVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: runLockVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: shmVolume,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: v1.HostShmPath,
				},
			},
		},
		{
			Name: tlsVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: desktop.GetName(),
				},
			},
		},
	}

	if t.IsUNIXDisplaySocket() && !strings.HasPrefix(path.Dir(t.GetDisplaySocketAddress()), v1.DesktopTmpPath) {
		volumes = append(volumes, corev1.Volume{
			Name: vncSockVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	if t.needsDedicatedPulseVolume() {
		volumes = append(volumes, corev1.Volume{
			Name: pulseSockVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// A PVC claim for the user if specified, otherwise use an EmptyDir.
	if cluster.GetUserdataVolumeSpec() != nil {
		volumes = append(volumes, corev1.Volume{
			Name: homeVolume,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: cluster.GetUserdataVolumeName(desktop.GetUser()),
				},
			},
		})
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: homeVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// If systemd we need to add a few more temp filesystems and bind mount
	// /sys/fs/cgroup.
	if t.GetInitSystem() == InitSystemd {
		volumes = append(volumes, []corev1.Volume{
			{
				Name: cgroupsVolume,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: v1.HostCgroupPath,
					},
				},
			},
		}...)
	}

	if additionalVolumes := t.GetVolumes(); additionalVolumes != nil {
		volumes = append(volumes, additionalVolumes...)
	}

	return volumes
}

// GetDesktopVolumeMounts returns the volume mounts for the main desktop container.
func (t *DesktopTemplate) GetDesktopVolumeMounts(cluster *VDICluster, desktop *Desktop) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      tmpVolume,
			MountPath: v1.DesktopTmpPath,
		},
		{
			Name:      runVolume,
			MountPath: v1.DesktopRunPath,
		},
		{
			Name:      runLockVolume,
			MountPath: v1.DesktopRunLockPath,
		},
		{
			Name:      shmVolume,
			MountPath: v1.DesktopShmPath,
		},
		{
			Name:      homeVolume,
			MountPath: fmt.Sprintf(v1.DesktopHomeFmt, desktop.GetUser()),
		},
	}
	if t.IsUNIXDisplaySocket() && !strings.HasPrefix(path.Dir(t.GetDisplaySocketAddress()), v1.DesktopTmpPath) {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      vncSockVolume,
			MountPath: filepath.Dir(t.GetDisplaySocketAddress()),
		})
	}
	if t.needsDedicatedPulseVolume() {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      pulseSockVolume,
			MountPath: filepath.Dir(t.GetPulseServer()),
		})
	}
	if t.GetInitSystem() == InitSystemd {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      cgroupsVolume,
			MountPath: v1.DesktopCgroupPath,
		})
	}
	if additionalMounts := t.GetVolumeMounts(); additionalMounts != nil {
		mounts = append(mounts, additionalMounts...)
	}
	return mounts
}

// GetDesktopProxyContainer returns the configuration for the kvdi-proxy sidecar.
func (t *DesktopTemplate) GetDesktopProxyContainer() corev1.Container {
	proxyVolMounts := []corev1.VolumeMount{
		{
			Name:      tmpVolume,
			MountPath: v1.DesktopTmpPath,
		},
		{
			Name:      runVolume,
			MountPath: v1.DesktopRunPath,
		},
		{
			Name:      runLockVolume,
			MountPath: v1.DesktopRunLockPath,
		},
		{
			Name:      tlsVolume,
			MountPath: v1.ServerCertificateMountPath,
			ReadOnly:  true,
		},
	}
	if t.IsUNIXDisplaySocket() && !strings.HasPrefix(path.Dir(t.GetDisplaySocketAddress()), v1.DesktopTmpPath) {
		proxyVolMounts = append(proxyVolMounts, corev1.VolumeMount{
			Name:      vncSockVolume,
			MountPath: filepath.Dir(t.GetDisplaySocketAddress()),
		})
	}
	if t.needsDedicatedPulseVolume() {
		proxyVolMounts = append(proxyVolMounts, corev1.VolumeMount{
			Name:      pulseSockVolume,
			MountPath: filepath.Dir(t.GetPulseServer()),
		})
	}
	if t.FileTransferEnabled() {
		proxyVolMounts = append(proxyVolMounts, corev1.VolumeMount{
			Name:      homeVolume,
			MountPath: v1.DesktopHomeMntPath,
		})
	}
	return corev1.Container{
		Name:            "kvdi-proxy",
		Image:           t.GetKVDIVNCProxyImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			"--vnc-addr", t.GetDisplaySocketURI(),
			"--user-id", strconv.Itoa(desktopUserID),
			"--pulse-server", t.GetPulseServer(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "web",
				ContainerPort: v1.WebPort,
			},
		},
		VolumeMounts: proxyVolMounts,
		// TODO: Make these configurable
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			// We need to be able to burst pretty high if the user wants to
			// download a large directory. An admin should be the one to determine
			// how many resources a user can use at any given time. This would also have
			// the benefit of limiting network traffic.
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
	}
}

// GetLifecycle returns the lifecycle actions for a desktop container booted from
// this template.
func (t *DesktopTemplate) GetLifecycle() *corev1.Lifecycle {
	if t.GetInitSystem() == InitSystemd {
		return &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"kill", "-s", "SIGRTMIN+3", "1"},
				},
			},
		}
	}
	return &corev1.Lifecycle{}
}
