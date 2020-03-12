package build_test

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/heroku/color"

	"github.com/apex/log"
	"github.com/docker/docker/api/types/container"

	"github.com/buildpacks/pack/internal/build/fakes"

	"github.com/docker/docker/client"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/build"
	ilogging "github.com/buildpacks/pack/internal/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

var (
	phasesRepoName string
)

func TestPhases(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	color.Disable(true)
	defer color.Disable(false)

	h.RequireDocker(t)

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	h.AssertNil(t, err)

	phasesRepoName = "phases.test.lc-" + h.RandString(10)

	wd, err := os.Getwd()
	h.AssertNil(t, err)

	// Create fake builder
	h.CreateImageFromDir(t, dockerCli, phasesRepoName, filepath.Join(wd, "testdata", "fake-lifecycle"))
	defer h.DockerRmi(dockerCli, phasesRepoName)

	spec.Run(t, "phases", testPhases, spec.Report(report.Terminal{}), spec.Sequential())
}

func testPhases(t *testing.T, when spec.G, it spec.S) {
	when("#Detect", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhase := &fakes.FakePhase{}
			fakePhaseFactory := fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Detect(context.Background(), "test", []string{}, fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			verboseLifecycle := fakeLifecycle(t, true)
			fakePhaseFactory := fakes.NewFakePhaseFactory()

			err := verboseLifecycle.Detect(context.Background(), "test", []string{"test"}, fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "detector")
			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
				[]string{"-app", "/workspace"},
				[]string{"-platform", "/platform"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhaseFactory := fakes.NewFakePhaseFactory()
			expectedNetworkMode := "some-network-mode"

			err := lifecycle.Detect(context.Background(), expectedNetworkMode, []string{}, fakePhaseFactory)
			h.AssertNil(t, err)

			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(expectedNetworkMode))
		})

		it("configures the phase with binds", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhaseFactory := fakes.NewFakePhaseFactory()
			expectedBinds := []string{"some-mount-source:/some-mount-target"}

			err := lifecycle.Detect(context.Background(), "test", expectedBinds, fakePhaseFactory)
			h.AssertNil(t, err)

			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertEq(t, configProvider.HostConfig().Binds, expectedBinds)
		})
	})

	when("#Restore", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhase := &fakes.FakePhase{}
			fakePhaseFactory := fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Restore(context.Background(), "test", fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with daemon access", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhaseFactory := fakes.NewFakePhaseFactory()

			err := lifecycle.Restore(context.Background(), "test", fakePhaseFactory)
			h.AssertNil(t, err)

			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertEq(t, configProvider.ContainerConfig().User, "root")
			h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, []string{"/var/run/docker.sock:/var/run/docker.sock"})
		})

		it("configures the phase with the expected arguments", func() {
			verboseLifecycle := fakeLifecycle(t, true)
			fakePhaseFactory := fakes.NewFakePhaseFactory()

			err := verboseLifecycle.Restore(context.Background(), "test", fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "restorer")
			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				[]string{"-log-level", "debug"},
				[]string{"-cache-dir", "/cache"},
				[]string{"-layers", "/layers"},
			)
		})

		it("configures the phase with binds", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhaseFactory := fakes.NewFakePhaseFactory()
			expectedBinds := []string{"some-cache:/cache"}

			err := lifecycle.Restore(context.Background(), "some-cache", fakePhaseFactory)
			h.AssertNil(t, err)

			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, expectedBinds)
		})
	})

	when("#Analyze", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhase := &fakes.FakePhase{}
			fakePhaseFactory := fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Analyze(context.Background(), "test", "test", false, false, fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		when("clear cache", func() {
			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, true, fakePhaseFactory)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "analyzer")
				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-skip-layers"},
				)
			})
		})

		when("clear cache is false", func() {
			it("configures the phase with the expected arguments", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepoName := "some-repo-name"

				err := lifecycle.Analyze(context.Background(), expectedRepoName, "test", false, false, fakePhaseFactory)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "analyzer")
				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-cache-dir", "/cache"},
				)
			})
		})

		when("publish", func() {
			it("configures the phase with registry access", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepos := []string{"some-repo-name"}

				err := lifecycle.Analyze(context.Background(), expectedRepos[0], "test", true, false, fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Env, []string{"CNB_REGISTRY_AUTH={}"})
				h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode("host"))
			})

			it("configures the phase with root", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()

				err := lifecycle.Analyze(context.Background(), "test", "test", true, false, fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
			})

			it("configures the phase with the expected arguments", func() {
				verboseLifecycle := fakeLifecycle(t, true)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepoName := "some-repo-name"

				err := verboseLifecycle.Analyze(context.Background(), expectedRepoName, "test", true, false, fakePhaseFactory)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "analyzer")
				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					//[]string{"-log-level", "debug"}, // TODO: fix [https://github.com/buildpacks/pack/issues/419].
					[]string{"-layers", "/layers"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedBinds := []string{"some-cache:/cache"}

				err := lifecycle.Analyze(context.Background(), "test", "some-cache", true, false, fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, expectedBinds)
			})
		})

		when("publish is false", func() {
			it("configures the phase with daemon access", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()

				err := lifecycle.Analyze(context.Background(), "test", "test", false, false, fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
				h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, []string{"/var/run/docker.sock:/var/run/docker.sock"})
			})

			it("configures the phase with the expected arguments", func() {
				verboseLifecycle := fakeLifecycle(t, true)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepoName := "some-repo-name"

				err := verboseLifecycle.Analyze(context.Background(), expectedRepoName, "test", false, true, fakePhaseFactory)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "analyzer")
				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{"-daemon"},
					[]string{"-layers", "/layers"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedBinds := []string{"some-cache:/cache"}

				err := lifecycle.Analyze(context.Background(), "test", "some-cache", false, true, fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, expectedBinds)
			})
		})
	})

	when("#Build", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhase := &fakes.FakePhase{}
			fakePhaseFactory := fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Build(context.Background(), "test", []string{}, fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		it("configures the phase with the expected arguments", func() {
			verboseLifecycle := fakeLifecycle(t, true)
			fakePhaseFactory := fakes.NewFakePhaseFactory()

			err := verboseLifecycle.Build(context.Background(), "test", []string{}, fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "builder")
			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertIncludeAllExpectedPatterns(t,
				configProvider.ContainerConfig().Cmd,
				//[]string{"-log-level", "debug"}, // TODO: fix [https://github.com/buildpacks/pack/issues/419].
				[]string{"-layers", "/layers"},
				[]string{"-app", "/workspace"},
				[]string{"-platform", "/platform"},
			)
		})

		it("configures the phase with the expected network mode", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhaseFactory := fakes.NewFakePhaseFactory()
			expectedNetworkMode := "some-network-mode"

			err := lifecycle.Build(context.Background(), expectedNetworkMode, []string{}, fakePhaseFactory)
			h.AssertNil(t, err)

			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode(expectedNetworkMode))
		})

		it("configures the phase with binds", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhaseFactory := fakes.NewFakePhaseFactory()
			expectedBinds := []string{"some-mount-source:/some-mount-target"}

			err := lifecycle.Build(context.Background(), "test", expectedBinds, fakePhaseFactory)
			h.AssertNil(t, err)

			configProvider := fakePhaseFactory.NewCalledWithProvider
			h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, expectedBinds)
		})
	})

	when("#Export", func() {
		it("creates a phase and then runs it", func() {
			lifecycle := fakeLifecycle(t, false)
			fakePhase := &fakes.FakePhase{}
			fakePhaseFactory := fakes.NewFakePhaseFactory(fakes.WhichReturnsForNew(fakePhase))

			err := lifecycle.Export(context.Background(), "test", "test", false, "test", "test", fakePhaseFactory)
			h.AssertNil(t, err)

			h.AssertEq(t, fakePhase.CleanupCallCount, 1)
			h.AssertEq(t, fakePhase.RunCallCount, 1)
		})

		when("publish", func() {
			it("configures the phase with registry access", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepos := []string{"some-repo-name", "some-run-image"}

				err := lifecycle.Export(context.Background(), expectedRepos[0], expectedRepos[1], true, "test", "test", fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t, configProvider.ContainerConfig().Env, []string{"CNB_REGISTRY_AUTH={}"})
				h.AssertEq(t, configProvider.HostConfig().NetworkMode, container.NetworkMode("host"))
			})

			it("configures the phase with the expected arguments", func() {
				verboseLifecycle := fakeLifecycle(t, true)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepoName := "some-repo-name"
				expectedRunImage := "some-run-image"
				expectedLaunchCacheName := "some-launch-cache"
				expectedCacheName := "some-cache"

				err := verboseLifecycle.Export(context.Background(), expectedRepoName, expectedRunImage, true, expectedLaunchCacheName, expectedCacheName, fakePhaseFactory)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "exporter")
				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{"-image", expectedRunImage},
					[]string{"-cache-dir", "/cache"},
					[]string{"-layers", "/layers"},
					[]string{"-app", "/workspace"},
					[]string{expectedRepoName},
				)
			})

			it("configures the phase with root", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()

				err := lifecycle.Export(context.Background(), "test", "test", true, "test", "test", fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedBinds := []string{"some-cache:/cache"}

				err := lifecycle.Export(context.Background(), "test", "test", true, "test", "some-cache", fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, expectedBinds)
			})
		})

		when("publish is false", func() {
			it("configures the phase with daemon access", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()

				err := lifecycle.Export(context.Background(), "test", "test", false, "test", "test", fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertEq(t, configProvider.ContainerConfig().User, "root")
				h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, []string{"/var/run/docker.sock:/var/run/docker.sock"})
			})

			it("configures the phase with the expected arguments", func() {
				verboseLifecycle := fakeLifecycle(t, true)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedRepoName := "some-repo-name"
				expectedRunImage := "some-run-image"
				expectedLaunchCacheName := "some-launch-cache"
				expectedCacheName := "some-cache"

				err := verboseLifecycle.Export(context.Background(), expectedRepoName, expectedRunImage, false, expectedLaunchCacheName, expectedCacheName, fakePhaseFactory)
				h.AssertNil(t, err)

				h.AssertEq(t, fakePhaseFactory.NewCalledWithName, "exporter")
				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t,
					configProvider.ContainerConfig().Cmd,
					[]string{"-log-level", "debug"},
					[]string{"-image", expectedRunImage},
					[]string{"-cache-dir", "/cache"},
					[]string{"-layers", "/layers"},
					[]string{"-app", "/workspace"},
					[]string{expectedRepoName},
					[]string{"-daemon"},
					[]string{"-launch-cache", "/launch-cache"},
				)
			})

			it("configures the phase with binds", func() {
				lifecycle := fakeLifecycle(t, false)
				fakePhaseFactory := fakes.NewFakePhaseFactory()
				expectedBinds := []string{"some-cache:/cache", "some-launch-cache:/launch-cache"}

				err := lifecycle.Export(context.Background(), "test", "test", false, "some-launch-cache", "some-cache", fakePhaseFactory)
				h.AssertNil(t, err)

				configProvider := fakePhaseFactory.NewCalledWithProvider
				h.AssertIncludeAllExpectedPatterns(t, configProvider.HostConfig().Binds, expectedBinds)
			})
		})
	})
}

func fakeLifecycle(t *testing.T, verbose bool) *build.Lifecycle {
	var outBuf bytes.Buffer
	logger := ilogging.NewLogWithWriters(&outBuf, &outBuf)
	if verbose {
		logger.Level = log.DebugLevel
	}

	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	h.AssertNil(t, err)

	lifecycle, err := CreateFakeLifecycle(filepath.Join("testdata", "fake-app"), docker, logger, phasesRepoName)
	h.AssertNil(t, err)

	return lifecycle
}
