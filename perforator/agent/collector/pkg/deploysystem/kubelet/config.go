package kubelet

type Config struct {
	TopologyLableKey             string   `yaml:"topology_lable_key,omitempty"`
	KubeletCgroupRoot            []string `yaml:"kubelet_cgroup_root,omitempty"`
	KubeletCgroupDriver          string   `yaml:"kubelet_cgroup_driver,omitempty"`
	KubeletCgroupQOSMode         string   `yaml:"kubelet_cgroup_qos_mode,omitempty"`
	KubeletCgroupContainerPrefix string   `yaml:"kubelet_cgroup_container_prefix,omitempty"`

	// See https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api
	KubernetesAPIServerHost string `yaml:"kubernetes_api_server_host,omitempty"`
}
