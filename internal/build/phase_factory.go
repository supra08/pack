package build

import (
	"fmt"
)

type DefaultPhaseFactory struct {
	lifecycle *Lifecycle
}

func NewDefaultPhaseFactory(lifecycle *Lifecycle) *DefaultPhaseFactory {
	return &DefaultPhaseFactory{lifecycle: lifecycle}
}

func (m *DefaultPhaseFactory) New(name string, provider *PhaseConfigProvider) (RunnerCleaner, error) {
	err := provider.Update(
		WithLifecycle(m.lifecycle),
		WithBinds([]string{
			fmt.Sprintf("%s:%s", m.lifecycle.LayersVolume, layersDir),
			fmt.Sprintf("%s:%s", m.lifecycle.AppVolume, appDir),
		}...),
	)
	if err != nil {
		return nil, err
	}

	phase := &Phase{
		ctrConf:  provider.ContainerConfig(),
		hostConf: provider.HostConfig(),
		name:     name,
		docker:   m.lifecycle.docker,
		logger:   m.lifecycle.logger,
		uid:      m.lifecycle.builder.UID,
		gid:      m.lifecycle.builder.GID,
		appPath:  m.lifecycle.appPath,
		appOnce:  m.lifecycle.appOnce,
	}

	return phase, nil
}
