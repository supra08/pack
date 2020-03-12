package build

import (
	"fmt"

	"github.com/buildpacks/lifecycle/auth"
	"github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
)

type PhaseConfigProviderOperation func(*PhaseConfigProvider) error

type PhaseConfigProvider struct {
	ctrConf  *container.Config
	hostConf *container.HostConfig
}

func NewPhaseConfigProvider(name string, ops ...PhaseConfigProviderOperation) (*PhaseConfigProvider, error) {
	provider := &PhaseConfigProvider{
		ctrConf:  new(container.Config),
		hostConf: new(container.HostConfig),
	}

	provider.ctrConf.Cmd = []string{"/cnb/lifecycle/" + name}
	provider.ctrConf.Labels = map[string]string{"author": "pack"}

	for _, op := range ops {
		if err := op(provider); err != nil {
			return nil, errors.Wrap(err, "create phase config")
		}
	}

	return provider, nil
}

func (p *PhaseConfigProvider) ContainerConfig() *container.Config {
	return p.ctrConf
}

func (p *PhaseConfigProvider) HostConfig() *container.HostConfig {
	return p.hostConf
}

func (p *PhaseConfigProvider) Update(ops ...PhaseConfigProviderOperation) error {
	for _, op := range ops {
		if err := op(p); err != nil {
			return errors.Wrap(err, "update phase config")
		}
	}
	return nil
}

func WithArgs(args ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		provider.ctrConf.Cmd = append(provider.ctrConf.Cmd, args...)
		return nil
	}
}

func WithBinds(binds ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		provider.hostConf.Binds = append(provider.hostConf.Binds, binds...)
		return nil
	}
}

func WithDaemonAccess() PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		provider.ctrConf.User = "root"
		provider.hostConf.Binds = append(provider.hostConf.Binds, "/var/run/docker.sock:/var/run/docker.sock")
		return nil
	}
}

func WithLifecycle(lifecycle *Lifecycle) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		provider.ctrConf.Image = lifecycle.builder.Name()

		if lifecycle.httpProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTP_PROXY="+lifecycle.httpProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "http_proxy="+lifecycle.httpProxy)
		}

		if lifecycle.httpsProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTPS_PROXY="+lifecycle.httpsProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "https_proxy="+lifecycle.httpsProxy)
		}

		if lifecycle.noProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "NO_PROXY="+lifecycle.noProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "no_proxy="+lifecycle.noProxy)
		}

		return nil
	}
}

func WithNetwork(networkMode string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		provider.hostConf.NetworkMode = container.NetworkMode(networkMode)
		return nil
	}
}

func WithRegistryAccess(repos ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repos...)
		if err != nil {
			return err
		}
		provider.ctrConf.Env = append(provider.ctrConf.Env, fmt.Sprintf(`CNB_REGISTRY_AUTH=%s`, authConfig))
		provider.hostConf.NetworkMode = container.NetworkMode("host")
		return nil
	}
}

func WithRoot() PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) error {
		provider.ctrConf.User = "root"
		return nil
	}
}
