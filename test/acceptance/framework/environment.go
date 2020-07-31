package framework

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultContextName   = "default"
	SecondaryContextName = "secondary"
)

// TestEnvironment represents the infrastructure environment of the test,
// such as the kubernetes cluster(s) the test is running against
type TestEnvironment interface {
	DefaultContext(t *testing.T) TestContext
	Context(t *testing.T, name string) TestContext
}

// TestContext represents a specific context a test needs,
// for example, information about a specific Kubernetes cluster.
type TestContext interface {
	Name() string
	KubectlOptions() *k8s.KubectlOptions
	KubernetesClient(t *testing.T) kubernetes.Interface
}

type kubernetesEnvironment struct {
	contexts map[string]*kubernetesContext
}

func newKubernetesEnvironmentFromConfig(config *TestConfig) *kubernetesEnvironment {
	defaultContext := NewContext(DefaultContextName, config.KubeNamespace, config.Kubeconfig, config.KubeContext)

	// Create a kubernetes environment with default context.
	kenv := &kubernetesEnvironment{
		contexts: map[string]*kubernetesContext{
			DefaultContextName: defaultContext,
		},
	}

	// Add secondary context if multi cluster tests are enabled.
	if config.EnableMultiCluster {
		kenv.contexts[SecondaryContextName] = NewContext(SecondaryContextName, config.SecondaryKubeNamespace, config.SecondaryKubeconfig, config.SecondaryKubeContext)
	}

	return kenv
}

func (k *kubernetesEnvironment) Context(t *testing.T, name string) TestContext {
	ctx, ok := k.contexts[name]
	require.Truef(t, ok, fmt.Sprintf("requested context %s not found", name))

	return ctx
}

func (k *kubernetesEnvironment) DefaultContext(t *testing.T) TestContext {
	ctx, ok := k.contexts[DefaultContextName]
	require.Truef(t, ok, "default context not found")

	return ctx
}

type kubernetesContext struct {
	name string

	pathToKubeConfig string
	kubeContextName  string
	namespace        string

	client kubernetes.Interface

	logDirectory string
}

func (k kubernetesContext) Name() string {
	return k.name
}

func (k kubernetesContext) KubectlOptions() *k8s.KubectlOptions {
	return &k8s.KubectlOptions{
		ContextName: k.kubeContextName,
		ConfigPath:  k.pathToKubeConfig,
		Namespace:   k.namespace,
	}
}

func (k kubernetesContext) KubernetesClient(t *testing.T) kubernetes.Interface {
	if k.client != nil {
		return k.client
	}

	configPath, err := k.KubectlOptions().GetConfigPath(t)
	require.NoError(t, err)

	config, err := k8s.LoadApiClientConfigE(configPath, k.kubeContextName)
	require.NoError(t, err)

	client, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	k.client = client

	return client
}

func NewContext(name, namespace, pathToKubeConfig, kubeContextName string) *kubernetesContext {
	return &kubernetesContext{
		name:             name,
		namespace:        namespace,
		pathToKubeConfig: pathToKubeConfig,
		kubeContextName:  kubeContextName,
	}
}
