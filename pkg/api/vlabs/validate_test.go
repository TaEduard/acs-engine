package vlabs

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/acs-engine/pkg/api/common"
	"github.com/Masterminds/semver"
)

const (
	ValidKubernetesNodeStatusUpdateFrequency        = "10s"
	ValidKubernetesCtrlMgrNodeMonitorGracePeriod    = "40s"
	ValidKubernetesCtrlMgrPodEvictionTimeout        = "5m0s"
	ValidKubernetesCtrlMgrRouteReconciliationPeriod = "10s"
	ValidKubernetesCloudProviderBackoff             = false
	ValidKubernetesCloudProviderBackoffRetries      = 6
	ValidKubernetesCloudProviderBackoffJitter       = 1
	ValidKubernetesCloudProviderBackoffDuration     = 5
	ValidKubernetesCloudProviderBackoffExponent     = 1.5
	ValidKubernetesCloudProviderRateLimit           = false
	ValidKubernetesCloudProviderRateLimitQPS        = 3
	ValidKubernetesCloudProviderRateLimitBucket     = 10
)

func Test_OrchestratorProfile_Validate(t *testing.T) {
	o := &OrchestratorProfile{
		OrchestratorType: "DCOS",
		KubernetesConfig: &KubernetesConfig{},
	}

	o.KubernetesConfig.ClusterSubnet = "10.0.0.0/16"
	if err := o.Validate(false); err == nil {
		t.Errorf("should error when KubernetesConfig populated for non-Kubernetes OrchestratorType")
	}

	o = &OrchestratorProfile{
		OrchestratorType: "Kubernetes",
		DcosConfig:       &DcosConfig{},
	}

	if err := o.Validate(false); err != nil {
		t.Errorf("should not error with empty object: %v", err)
	}

	o.DcosConfig.DcosWindowsBootstrapURL = "http://www.microsoft.com"
	if err := o.Validate(false); err == nil {
		t.Errorf("should error when DcosConfig populated for non-Kubernetes OrchestratorType")
	}

	o.DcosConfig.DcosBootstrapURL = "http://www.microsoft.com"
	if err := o.Validate(false); err == nil {
		t.Errorf("should error when DcosConfig populated for non-Kubernetes OrchestratorType")
	}

	o = &OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: "1.7.3",
	}

	if err := o.Validate(false); err == nil {
		t.Errorf("should have failed on old patch version")
	}

	if err := o.Validate(true); err != nil {
		t.Errorf("should not have failed on old patch version during update valdiation")
	}

	o = &OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: "v1.9.0",
	}

	if err := o.Validate(false); err != nil {
		t.Errorf("should not have failed on version with v prefix")
	}

	o = &OrchestratorProfile{
		OrchestratorType:    OpenShift,
		OrchestratorVersion: "v1.0",
	}

	if err := o.Validate(false); err == nil {
		t.Errorf("should have failed on old version")
	}
	if err := o.Validate(true); err != nil {
		t.Errorf("should not have failed on old version")
	}

	o = &OrchestratorProfile{
		OrchestratorType:    Kubernetes,
		OrchestratorVersion: "v1.9.0",
		OpenShiftConfig:     &OpenShiftConfig{},
	}
	if err := o.Validate(false); err == nil {
		t.Errorf("should have failed on OpenShift config specified with non OpenShift orchestrator type")
	}
}

func Test_KubernetesConfig_Validate(t *testing.T) {
	// Tests that should pass across all versions
	for _, k8sVersion := range common.GetAllSupportedKubernetesVersions() {
		c := KubernetesConfig{}
		if err := c.Validate(k8sVersion); err != nil {
			t.Errorf("should not error on empty KubernetesConfig: %v, version %s", err, k8sVersion)
		}

		c = KubernetesConfig{
			ClusterSubnet:                "10.120.0.0/16",
			DockerBridgeSubnet:           "10.120.1.0/16",
			MaxPods:                      42,
			CloudProviderBackoff:         ValidKubernetesCloudProviderBackoff,
			CloudProviderBackoffRetries:  ValidKubernetesCloudProviderBackoffRetries,
			CloudProviderBackoffJitter:   ValidKubernetesCloudProviderBackoffJitter,
			CloudProviderBackoffDuration: ValidKubernetesCloudProviderBackoffDuration,
			CloudProviderBackoffExponent: ValidKubernetesCloudProviderBackoffExponent,
			CloudProviderRateLimit:       ValidKubernetesCloudProviderRateLimit,
			CloudProviderRateLimitQPS:    ValidKubernetesCloudProviderRateLimitQPS,
			CloudProviderRateLimitBucket: ValidKubernetesCloudProviderRateLimitBucket,
			KubeletConfig: map[string]string{
				"--node-status-update-frequency": ValidKubernetesNodeStatusUpdateFrequency,
			},
			ControllerManagerConfig: map[string]string{
				"--node-monitor-grace-period":   ValidKubernetesCtrlMgrNodeMonitorGracePeriod,
				"--pod-eviction-timeout":        ValidKubernetesCtrlMgrPodEvictionTimeout,
				"--route-reconciliation-period": ValidKubernetesCtrlMgrRouteReconciliationPeriod,
			},
		}
		if err := c.Validate(k8sVersion); err != nil {
			t.Errorf("should not error on a KubernetesConfig with valid param values: %v", err)
		}

		c = KubernetesConfig{
			ClusterSubnet: "10.16.x.0/invalid",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid ClusterSubnet")
		}

		c = KubernetesConfig{
			DockerBridgeSubnet: "10.120.1.0/invalid",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid DockerBridgeSubnet")
		}

		c = KubernetesConfig{
			KubeletConfig: map[string]string{
				"--non-masquerade-cidr": "10.120.1.0/24",
			},
		}
		if err := c.Validate(k8sVersion); err != nil {
			t.Error("should not error on valid --non-masquerade-cidr")
		}

		c = KubernetesConfig{
			KubeletConfig: map[string]string{
				"--non-masquerade-cidr": "10.120.1.0/invalid",
			},
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid --non-masquerade-cidr")
		}

		c = KubernetesConfig{
			MaxPods: KubernetesMinMaxPods - 1,
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid MaxPods")
		}

		c = KubernetesConfig{
			KubeletConfig: map[string]string{
				"--node-status-update-frequency": "invalid",
			},
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid --node-status-update-frequency kubelet config")
		}

		c = KubernetesConfig{
			ControllerManagerConfig: map[string]string{
				"--node-monitor-grace-period": "invalid",
			},
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid --node-monitor-grace-period")
		}

		c = KubernetesConfig{
			ControllerManagerConfig: map[string]string{
				"--node-monitor-grace-period": "30s",
			},
			KubeletConfig: map[string]string{
				"--node-status-update-frequency": "10s",
			},
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when --node-monitor-grace-period is not sufficiently larger than --node-status-update-frequency kubelet config")
		}

		c = KubernetesConfig{
			ControllerManagerConfig: map[string]string{
				"--pod-eviction-timeout": "invalid",
			},
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid --pod-eviction-timeout")
		}

		c = KubernetesConfig{
			ControllerManagerConfig: map[string]string{
				"--route-reconciliation-period": "invalid",
			},
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error on invalid --route-reconciliation-period")
		}

		c = KubernetesConfig{
			DNSServiceIP: "192.168.0.10",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when DNSServiceIP but not ServiceCidr")
		}

		c = KubernetesConfig{
			ServiceCidr: "192.168.0.10/24",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when ServiceCidr but not DNSServiceIP")
		}

		c = KubernetesConfig{
			DNSServiceIP: "invalid",
			ServiceCidr:  "192.168.0.0/24",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when DNSServiceIP is invalid")
		}

		c = KubernetesConfig{
			DNSServiceIP: "192.168.1.10",
			ServiceCidr:  "192.168.0.0/not-a-len",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when ServiceCidr is invalid")
		}

		c = KubernetesConfig{
			DNSServiceIP: "192.168.1.10",
			ServiceCidr:  "192.168.0.0/24",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when DNSServiceIP is outside of ServiceCidr")
		}

		c = KubernetesConfig{
			DNSServiceIP: "172.99.255.255",
			ServiceCidr:  "172.99.0.1/16",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when DNSServiceIP is broadcast address of ServiceCidr")
		}

		c = KubernetesConfig{
			DNSServiceIP: "172.99.0.1",
			ServiceCidr:  "172.99.0.1/16",
		}
		if err := c.Validate(k8sVersion); err == nil {
			t.Error("should error when DNSServiceIP is first IP of ServiceCidr")
		}

		c = KubernetesConfig{
			DNSServiceIP: "172.99.255.10",
			ServiceCidr:  "172.99.0.1/16",
		}
		if err := c.Validate(k8sVersion); err != nil {
			t.Error("should not error when DNSServiceIP and ServiceCidr are valid")
		}
	}

	// Tests that apply to 1.6 and later releases
	for _, k8sVersion := range common.GetAllSupportedKubernetesVersions() {
		c := KubernetesConfig{
			CloudProviderBackoff:   true,
			CloudProviderRateLimit: true,
		}
		if err := c.Validate(k8sVersion); err != nil {
			t.Error("should not error when basic backoff and rate limiting are set to true with no options")
		}
	}

	trueVal := true
	// Tests that apply to 1.8 and later releases
	for _, k8sVersion := range common.GetVersionsGt(common.GetAllSupportedKubernetesVersions(), "1.8.0", true, true) {
		c := KubernetesConfig{
			UseCloudControllerManager: &trueVal,
		}
		if err := c.Validate(k8sVersion); err != nil {
			t.Error("should not error because UseCloudControllerManager is available since v1.8")
		}
	}
}

func Test_Properties_ValidateNetworkPolicy(t *testing.T) {
	p := &Properties{}
	p.OrchestratorProfile = &OrchestratorProfile{}
	p.OrchestratorProfile.OrchestratorType = Kubernetes

	for _, policy := range NetworkPolicyValues {
		p.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{}
		p.OrchestratorProfile.KubernetesConfig.NetworkPolicy = policy
		if err := p.validateNetworkPolicy(); err != nil {
			t.Errorf(
				"should not error on networkPolicy=\"%s\"",
				policy,
			)
		}
	}

	p.OrchestratorProfile.KubernetesConfig.NetworkPolicy = "not-existing"
	if err := p.validateNetworkPolicy(); err == nil {
		t.Errorf(
			"should error on invalid networkPolicy",
		)
	}

	p.OrchestratorProfile.KubernetesConfig.NetworkPolicy = "calico"
	p.AgentPoolProfiles = []*AgentPoolProfile{
		{
			OSType: Windows,
		},
	}
	if err := p.validateNetworkPolicy(); err == nil {
		t.Errorf(
			"should error on calico for windows clusters",
		)
	}

	p.OrchestratorProfile.KubernetesConfig.NetworkPolicy = "cilium"
	p.AgentPoolProfiles = []*AgentPoolProfile{
		{
			OSType: Windows,
		},
	}
	if err := p.validateNetworkPolicy(); err == nil {
		t.Errorf(
			"should error on cilium for windows clusters",
		)
	}
}

func Test_Properties_ValidateNetworkPlugin(t *testing.T) {
	p := &Properties{}
	p.OrchestratorProfile = &OrchestratorProfile{}
	p.OrchestratorProfile.OrchestratorType = Kubernetes

	for _, policy := range NetworkPluginValues {
		p.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{}
		p.OrchestratorProfile.KubernetesConfig.NetworkPlugin = policy
		if err := p.validateNetworkPlugin(); err != nil {
			t.Errorf(
				"should not error on networkPolicy=\"%s\"",
				policy,
			)
		}
	}

	p.OrchestratorProfile.KubernetesConfig.NetworkPlugin = "not-existing"
	if err := p.validateNetworkPlugin(); err == nil {
		t.Errorf(
			"should error on invalid networkPlugin",
		)
	}
}

func Test_Properties_ValidateNetworkPluginPlusPolicy(t *testing.T) {
	p := &Properties{}
	p.OrchestratorProfile = &OrchestratorProfile{}
	p.OrchestratorProfile.OrchestratorType = Kubernetes

	for _, config := range networkPluginPlusPolicyAllowed {
		p.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{}
		p.OrchestratorProfile.KubernetesConfig.NetworkPlugin = config.networkPlugin
		p.OrchestratorProfile.KubernetesConfig.NetworkPolicy = config.networkPolicy
		if err := p.validateNetworkPluginPlusPolicy(); err != nil {
			t.Errorf(
				"should not error on networkPolicy=\"%s\" + networkPlugin=\"%s\"",
				config.networkPolicy, config.networkPlugin,
			)
		}
	}

	for _, config := range []k8sNetworkConfig{
		{
			networkPlugin: "azure",
			networkPolicy: "calico",
		},
		{
			networkPlugin: "azure",
			networkPolicy: "cilium",
		},
		{
			networkPlugin: "azure",
			networkPolicy: "azure",
		},
		{
			networkPlugin: "kubenet",
			networkPolicy: "none",
		},
		{
			networkPlugin: "azure",
			networkPolicy: "none",
		},
		{
			networkPlugin: "kubenet",
			networkPolicy: "kubenet",
		},
	} {
		p.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{}
		p.OrchestratorProfile.KubernetesConfig.NetworkPlugin = config.networkPlugin
		p.OrchestratorProfile.KubernetesConfig.NetworkPolicy = config.networkPolicy
		if err := p.validateNetworkPluginPlusPolicy(); err == nil {
			t.Errorf(
				"should error on networkPolicy=\"%s\" + networkPlugin=\"%s\"",
				config.networkPolicy, config.networkPlugin,
			)
		}
	}
}

func Test_ServicePrincipalProfile_ValidateSecretOrKeyvaultSecretRef(t *testing.T) {

	t.Run("ServicePrincipalProfile with secret should pass", func(t *testing.T) {
		p := getK8sDefaultProperties(false)

		if err := p.Validate(false); err != nil {
			t.Errorf("should not error %v", err)
		}
	})

	t.Run("ServicePrincipalProfile with KeyvaultSecretRef (with version) should pass", func(t *testing.T) {
		p := getK8sDefaultProperties(false)
		p.ServicePrincipalProfile.Secret = ""
		p.ServicePrincipalProfile.KeyvaultSecretRef = &KeyvaultSecretRef{
			VaultID:       "/subscriptions/SUB-ID/resourceGroups/RG-NAME/providers/Microsoft.KeyVault/vaults/KV-NAME",
			SecretName:    "secret-name",
			SecretVersion: "version",
		}
		if err := p.Validate(false); err != nil {
			t.Errorf("should not error %v", err)
		}
	})

	t.Run("ServicePrincipalProfile with KeyvaultSecretRef (without version) should pass", func(t *testing.T) {
		p := getK8sDefaultProperties(false)
		p.ServicePrincipalProfile.Secret = ""
		p.ServicePrincipalProfile.KeyvaultSecretRef = &KeyvaultSecretRef{
			VaultID:    "/subscriptions/SUB-ID/resourceGroups/RG-NAME/providers/Microsoft.KeyVault/vaults/KV-NAME",
			SecretName: "secret-name",
		}

		if err := p.Validate(false); err != nil {
			t.Errorf("should not error %v", err)
		}
	})

	t.Run("ServicePrincipalProfile with Secret and KeyvaultSecretRef should NOT pass", func(t *testing.T) {
		p := getK8sDefaultProperties(false)
		p.ServicePrincipalProfile.Secret = "secret"
		p.ServicePrincipalProfile.KeyvaultSecretRef = &KeyvaultSecretRef{
			VaultID:    "/subscriptions/SUB-ID/resourceGroups/RG-NAME/providers/Microsoft.KeyVault/vaults/KV-NAME",
			SecretName: "secret-name",
		}

		if err := p.Validate(false); err == nil {
			t.Error("error should have occurred")
		}
	})

	t.Run("ServicePrincipalProfile with incorrect KeyvaultSecretRef format should NOT pass", func(t *testing.T) {
		p := getK8sDefaultProperties(false)
		p.ServicePrincipalProfile.Secret = ""
		p.ServicePrincipalProfile.KeyvaultSecretRef = &KeyvaultSecretRef{
			VaultID:    "randomID",
			SecretName: "secret-name",
		}

		if err := p.Validate(false); err == nil || err.Error() != "service principal client keyvault secret reference is of incorrect format" {
			t.Error("error should have occurred")
		}
	})
}

func TestValidateKubernetesLabelValue(t *testing.T) {

	validLabelValues := []string{"", "a", "a1", "this--valid--label--is--exactly--sixty--three--characters--long", "123456", "my-label_valid.com"}
	invalidLabelValues := []string{"a$$b", "-abc", "not.valid.", "This____long____label___is______sixty______four_____chararacters", "Label with spaces"}

	for _, l := range validLabelValues {
		if err := validateKubernetesLabelValue(l); err != nil {
			t.Fatalf("Label value %v should not return error: %v", l, err)
		}
	}

	for _, l := range invalidLabelValues {
		if err := validateKubernetesLabelValue(l); err == nil {
			t.Fatalf("Label value %v should return an error", l)
		}
	}
}

func TestValidateKubernetesLabelKey(t *testing.T) {

	validLabelKeys := []string{"a", "a1", "this--valid--label--is--exactly--sixty--three--characters--long", "123456", "my-label_valid.com", "foo.bar/name", "1.2321.324/key_name.foo", "valid.long.253.characters.label.key.prefix.12345678910.fooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooo/my-key"}
	invalidLabelKeys := []string{"", "a/b/c", ".startswithdot", "spaces in key", "foo/", "/name", "$.$/com", "too-long-254-characters-key-prefix-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------123/name", "wrong-slash\\foo"}

	for _, l := range validLabelKeys {
		if err := validateKubernetesLabelKey(l); err != nil {
			t.Fatalf("Label key %v should not return error: %v", l, err)
		}
	}

	for _, l := range invalidLabelKeys {
		if err := validateKubernetesLabelKey(l); err == nil {
			t.Fatalf("Label key %v should return an error", l)
		}
	}
}

func Test_AadProfile_Validate(t *testing.T) {
	t.Run("Valid aadProfile should pass", func(t *testing.T) {
		for _, aadProfile := range []AADProfile{
			{
				ClientAppID: "92444486-5bc3-4291-818b-d53ae480991b",
				ServerAppID: "403f018b-4d89-495b-b548-0cf9868cdb0a",
			},
			{
				ClientAppID: "92444486-5bc3-4291-818b-d53ae480991b",
				ServerAppID: "403f018b-4d89-495b-b548-0cf9868cdb0a",
				TenantID:    "feb784f6-7174-46da-aeae-da66e80c7a11",
			},
		} {
			if err := aadProfile.Validate(); err != nil {
				t.Errorf("should not error %v", err)
			}
		}
	})

	t.Run("Invalid aadProfiles should NOT pass", func(t *testing.T) {
		for _, aadProfile := range []AADProfile{
			{
				ClientAppID: "1",
				ServerAppID: "d",
			},
			{
				ClientAppID: "6a247d73-ae33-4559-8e5d-4001fdc17b15",
			},
			{
				ClientAppID: "92444486-5bc3-4291-818b-d53ae480991b",
				ServerAppID: "403f018b-4d89-495b-b548-0cf9868cdb0a",
				TenantID:    "1",
			},
			{},
		} {
			if err := aadProfile.Validate(); err == nil {
				t.Errorf("error should have occurred")
			}
		}
	})
}

func getK8sDefaultProperties(hasWindows bool) *Properties {
	p := &Properties{
		OrchestratorProfile: &OrchestratorProfile{
			OrchestratorType: Kubernetes,
		},
		MasterProfile: &MasterProfile{
			Count:     1,
			DNSPrefix: "foo",
			VMSize:    "Standard_DS2_v2",
		},
		AgentPoolProfiles: []*AgentPoolProfile{
			{
				Name:                "agentpool",
				VMSize:              "Standard_D2_v2",
				Count:               1,
				AvailabilityProfile: AvailabilitySet,
			},
		},
		LinuxProfile: &LinuxProfile{
			AdminUsername: "azureuser",
			SSH: struct {
				PublicKeys []PublicKey `json:"publicKeys" validate:"required,len=1"`
			}{
				PublicKeys: []PublicKey{{
					KeyData: "publickeydata",
				}},
			},
		},
		ServicePrincipalProfile: &ServicePrincipalProfile{
			ClientID: "clientID",
			Secret:   "clientSecret",
		},
	}

	if hasWindows {
		p.AgentPoolProfiles = []*AgentPoolProfile{
			{
				Name:                "agentpool",
				VMSize:              "Standard_D2_v2",
				Count:               1,
				AvailabilityProfile: AvailabilitySet,
				OSType:              Windows,
			},
		}
		p.WindowsProfile = &WindowsProfile{
			AdminUsername: "azureuser",
			AdminPassword: "password",
		}
	}

	return p
}

func Test_Properties_ValidateContainerRuntime(t *testing.T) {
	p := &Properties{}
	p.OrchestratorProfile = &OrchestratorProfile{}
	p.OrchestratorProfile.OrchestratorType = Kubernetes

	for _, runtime := range ContainerRuntimeValues {
		p.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{}
		p.OrchestratorProfile.KubernetesConfig.ContainerRuntime = runtime
		if err := p.validateContainerRuntime(); err != nil {
			t.Errorf(
				"should not error on containerRuntime=\"%s\"",
				runtime,
			)
		}
	}

	p.OrchestratorProfile.KubernetesConfig.ContainerRuntime = "not-existing"
	if err := p.validateContainerRuntime(); err == nil {
		t.Errorf(
			"should error on invalid containerRuntime",
		)
	}

	p.OrchestratorProfile.KubernetesConfig.ContainerRuntime = "clear-containers"
	p.AgentPoolProfiles = []*AgentPoolProfile{
		{
			OSType: Windows,
		},
	}
	if err := p.validateContainerRuntime(); err == nil {
		t.Errorf(
			"should error on clear-containers for windows clusters",
		)
	}
}

func TestWindowsVersions(t *testing.T) {
	for _, version := range common.GetAllSupportedKubernetesVersionsWindows() {
		p := getK8sDefaultProperties(true)
		p.OrchestratorProfile.OrchestratorVersion = version
		if err := p.Validate(false); err != nil {
			t.Errorf(
				"should not error on valid Windows version: %v", err,
			)
		}
		sv, _ := semver.NewVersion(version)
		p = getK8sDefaultProperties(true)
		p.OrchestratorProfile.OrchestratorRelease = fmt.Sprintf("%d.%d", sv.Major(), sv.Minor())
		if err := p.Validate(false); err != nil {
			t.Errorf(
				"should not error on valid Windows version: %v", err,
			)
		}
	}
	p := getK8sDefaultProperties(true)
	p.OrchestratorProfile.OrchestratorRelease = "1.4"
	if err := p.Validate(false); err == nil {
		t.Errorf(
			"should error on invalid Windows version",
		)
	}

	p = getK8sDefaultProperties(true)
	p.OrchestratorProfile.OrchestratorVersion = "1.4.0"
	if err := p.Validate(false); err == nil {
		t.Errorf(
			"should error on invalid Windows version",
		)
	}
}

func TestLinuxVersions(t *testing.T) {
	for _, version := range common.GetAllSupportedKubernetesVersions() {
		p := getK8sDefaultProperties(false)
		p.OrchestratorProfile.OrchestratorVersion = version
		if err := p.Validate(false); err != nil {
			t.Errorf(
				"should not error on valid Linux version: %v", err,
			)
		}
		sv, _ := semver.NewVersion(version)
		p = getK8sDefaultProperties(false)
		p.OrchestratorProfile.OrchestratorRelease = fmt.Sprintf("%d.%d", sv.Major(), sv.Minor())
		if err := p.Validate(false); err != nil {
			t.Errorf(
				"should not error on valid Linux version: %v", err,
			)
		}
	}
	p := getK8sDefaultProperties(false)
	p.OrchestratorProfile.OrchestratorRelease = "1.4"
	if err := p.Validate(false); err == nil {
		t.Errorf(
			"should error on invalid Linux version",
		)
	}

	p = getK8sDefaultProperties(false)
	p.OrchestratorProfile.OrchestratorVersion = "1.4.0"
	if err := p.Validate(false); err == nil {
		t.Errorf(
			"should error on invalid Linux version",
		)
	}
}

func TestValidateImageNameAndGroup(t *testing.T) {
	tests := []struct {
		name string

		imageName          string
		imageResourceGroup string

		expectedErr error
	}{
		{
			name: "valid run",

			imageName:          "rhel9000",
			imageResourceGroup: "club",

			expectedErr: nil,
		},
		{
			name: "invalid: image name is missing",

			imageResourceGroup: "club",

			expectedErr: errors.New(`imageName needs to be specified when imageResourceGroup is provided`),
		},
		{
			name: "invalid: image resource group is missing",

			imageName: "rhel9000",

			expectedErr: errors.New(`imageResourceGroup needs to be specified when imageName is provided`),
		},
	}

	for _, test := range tests {
		t.Logf("scenario %q", test.name)

		gotErr := validateImageNameAndGroup(test.imageName, test.imageResourceGroup)
		if !reflect.DeepEqual(gotErr, test.expectedErr) {
			t.Errorf("expected error: %v, got: %v", test.expectedErr, gotErr)
		}
	}
}

func TestOpenshiftValidate(t *testing.T) {
	tests := []struct {
		name string

		properties *Properties
		isUpgrade  bool

		expectedErr error
	}{
		{
			name: "valid",

			properties: &Properties{
				AzProfile: &AzProfile{
					Location:       "eastus",
					ResourceGroup:  "group",
					SubscriptionID: "sub_id",
					TenantID:       "tenant_id",
				},
				OrchestratorProfile: &OrchestratorProfile{
					OrchestratorType: OpenShift,
					OpenShiftConfig: &OpenShiftConfig{
						ClusterUsername: "user",
						ClusterPassword: "pass",
					},
				},
				MasterProfile: &MasterProfile{
					Count:          1,
					DNSPrefix:      "mydns",
					VMSize:         "Standard_D4s_v3",
					StorageProfile: ManagedDisks,
				},
				AgentPoolProfiles: []*AgentPoolProfile{
					{
						Name:                "compute",
						Count:               1,
						VMSize:              "Standard_D4s_v3",
						StorageProfile:      ManagedDisks,
						AvailabilityProfile: AvailabilitySet,
					},
				},
				LinuxProfile: &LinuxProfile{
					AdminUsername: "admin",
					SSH: struct {
						PublicKeys []PublicKey `json:"publicKeys" validate:"required,len=1"`
					}{
						PublicKeys: []PublicKey{
							{KeyData: "ssh-key"},
						},
					},
				},
			},
			isUpgrade: false,

			expectedErr: nil,
		},
		{
			name: "invalid - masterProfile.storageProfile needs to be ManagedDisks",

			properties: &Properties{
				AzProfile: &AzProfile{
					Location:       "eastus",
					ResourceGroup:  "group",
					SubscriptionID: "sub_id",
					TenantID:       "tenant_id",
				},
				OrchestratorProfile: &OrchestratorProfile{
					OrchestratorType: OpenShift,
					OpenShiftConfig: &OpenShiftConfig{
						ClusterUsername: "user",
						ClusterPassword: "pass",
					},
				},
				MasterProfile: &MasterProfile{
					Count:          1,
					DNSPrefix:      "mydns",
					VMSize:         "Standard_D4s_v3",
					StorageProfile: StorageAccount,
				},
				LinuxProfile: &LinuxProfile{
					AdminUsername: "admin",
					SSH: struct {
						PublicKeys []PublicKey `json:"publicKeys" validate:"required,len=1"`
					}{
						PublicKeys: []PublicKey{
							{KeyData: "ssh-key"},
						},
					},
				},
			},
			isUpgrade: false,

			expectedErr: errors.New("OpenShift orchestrator supports only ManagedDisks"),
		},
		{
			name: "invalid - agentPoolProfile[0].storageProfile needs to be ManagedDisks",

			properties: &Properties{
				AzProfile: &AzProfile{
					Location:       "eastus",
					ResourceGroup:  "group",
					SubscriptionID: "sub_id",
					TenantID:       "tenant_id",
				},
				OrchestratorProfile: &OrchestratorProfile{
					OrchestratorType: OpenShift,
					OpenShiftConfig: &OpenShiftConfig{
						ClusterUsername: "user",
						ClusterPassword: "pass",
					},
				},
				MasterProfile: &MasterProfile{
					Count:          1,
					DNSPrefix:      "mydns",
					VMSize:         "Standard_D4s_v3",
					StorageProfile: ManagedDisks,
				},
				AgentPoolProfiles: []*AgentPoolProfile{
					{
						Name:                "compute",
						Count:               1,
						VMSize:              "Standard_D4s_v3",
						StorageProfile:      StorageAccount,
						AvailabilityProfile: AvailabilitySet,
					},
				},
				LinuxProfile: &LinuxProfile{
					AdminUsername: "admin",
					SSH: struct {
						PublicKeys []PublicKey `json:"publicKeys" validate:"required,len=1"`
					}{
						PublicKeys: []PublicKey{
							{KeyData: "ssh-key"},
						},
					},
				},
			},
			isUpgrade: false,

			expectedErr: errors.New("OpenShift orchestrator supports only ManagedDisks"),
		},
	}

	for _, test := range tests {
		t.Logf("running scenario %q", test.name)

		gotErr := test.properties.Validate(test.isUpgrade)
		if !reflect.DeepEqual(test.expectedErr, gotErr) {
			t.Errorf("expected error: %v\ngot error: %v", test.expectedErr, gotErr)
		}
	}
}
