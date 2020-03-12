package build

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
)

const (
	layersDir      = "/layers"
	appDir         = "/workspace"
	cacheDir       = "/cache"
	launchCacheDir = "/launch-cache"
	platformDir    = "/platform"
)

type RunnerCleaner interface {
	Run(ctx context.Context) error
	Cleanup() error
}

type PhaseFactory interface {
	New(name string, pcp *PhaseConfigProvider) (RunnerCleaner, error)
}

func (l *Lifecycle) Detect(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	phaseName := "detector"

	configProvider, err := NewPhaseConfigProvider(
		phaseName,
		WithArgs(
			l.withLogLevel(
				"-app", appDir,
				"-platform", platformDir,
			)...,
		),
		WithNetwork(networkMode),
		WithBinds(volumes...),
	)
	if err != nil {
		return err
	}

	detect, err := phaseFactory.New(
		phaseName,
		configProvider,
	)
	if err != nil {
		return err
	}

	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *Lifecycle) Restore(ctx context.Context, cacheName string, phaseFactory PhaseFactory) error {
	phaseName := "restorer"

	configProvider, err := NewPhaseConfigProvider(
		phaseName,
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(
				"-cache-dir", cacheDir,
				"-layers", layersDir,
			)...,
		),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
	if err != nil {
		return err
	}

	restore, err := phaseFactory.New(
		phaseName,
		configProvider,
	)
	if err != nil {
		return err
	}

	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *Lifecycle) Analyze(ctx context.Context, repoName, cacheName string, publish, clearCache bool, phaseFactory PhaseFactory) error {
	analyze, err := l.newAnalyze(repoName, cacheName, publish, clearCache, phaseFactory)
	if err != nil {
		return err
	}
	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *Lifecycle) newAnalyze(repoName, cacheName string, publish, clearCache bool, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	args := []string{
		"-layers", layersDir,
		repoName,
	}
	if clearCache {
		args = prependArg("-skip-layers", args)
	} else {
		args = append([]string{"-cache-dir", cacheDir}, args...)
	}

	phaseName := "analyzer"

	if publish {
		configProvider, err := NewPhaseConfigProvider(
			phaseName,
			WithRegistryAccess(repoName),
			WithRoot(),
			WithArgs(args...),
			WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
		)
		if err != nil {
			return nil, err
		}

		return phaseFactory.New(
			phaseName,
			configProvider,
		)
	}

	configProvider, err := NewPhaseConfigProvider(
		phaseName,
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(
				prependArg(
					"-daemon",
					args,
				)...,
			)...,
		),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
	if err != nil {
		return nil, err
	}

	return phaseFactory.New(
		phaseName,
		configProvider,
	)
}

func prependArg(arg string, args []string) []string {
	return append([]string{arg}, args...)
}

func (l *Lifecycle) Build(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	phaseName := "builder"

	configProvider, err := NewPhaseConfigProvider(
		phaseName,
		WithArgs(
			"-layers", layersDir,
			"-app", appDir,
			"-platform", platformDir,
		),
		WithNetwork(networkMode),
		WithBinds(volumes...),
	)
	if err != nil {
		return err
	}

	build, err := phaseFactory.New(
		phaseName,
		configProvider,
	)
	if err != nil {
		return err
	}

	defer build.Cleanup()
	return build.Run(ctx)
}

func (l *Lifecycle) Export(ctx context.Context, repoName string, runImage string, publish bool, launchCacheName, cacheName string, phaseFactory PhaseFactory) error {
	export, err := l.newExport(repoName, runImage, publish, launchCacheName, cacheName, phaseFactory)
	if err != nil {
		return err
	}
	defer export.Cleanup()
	return export.Run(ctx)
}

func (l *Lifecycle) newExport(repoName, runImage string, publish bool, launchCacheName, cacheName string, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	args := []string{
		"-image", runImage,
		"-cache-dir", cacheDir,
		"-layers", layersDir,
		"-app", appDir,
		repoName,
	}

	binds := []string{fmt.Sprintf("%s:%s", cacheName, cacheDir)}

	phaseName := "exporter"

	if publish {
		configProvider, err := NewPhaseConfigProvider(
			phaseName,
			WithRegistryAccess(repoName, runImage),
			WithArgs(
				l.withLogLevel(args...)...,
			),
			WithRoot(),
			WithBinds(binds...),
		)
		if err != nil {
			return nil, err
		}

		return phaseFactory.New(
			phaseName,
			configProvider,
		)
	}

	args = append([]string{"-daemon", "-launch-cache", launchCacheDir}, args...)
	binds = append(binds, fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir))

	configProvider, err := NewPhaseConfigProvider(
		phaseName,
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(args...)...,
		),
		WithBinds(binds...),
	)
	if err != nil {
		return nil, err
	}

	return phaseFactory.New(
		phaseName,
		configProvider,
	)
}

func (l *Lifecycle) withLogLevel(args ...string) []string {
	version := semver.MustParse(l.version)
	if semver.MustParse("0.4.0").LessThan(version) {
		if l.logger.IsVerbose() {
			return append([]string{"-log-level", "debug"}, args...)
		}
	}
	return args
}
