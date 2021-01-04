/*
Copyright 2021 Codelogia

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bazelbuild/rules_docker/container/go/pkg/compat"
	"github.com/spf13/cobra"
)

// New constructs a new package command.
func New() *cobra.Command {
	var helmPath string
	var packageDir string
	var output string
	var version string
	var bazelInfoFile, bazelVersionFile string

	cmd := &cobra.Command{
		Use:   "package",
		Short: "Packages a Helm chart.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			stamper, err := compat.NewStamper([]string{bazelInfoFile, bazelVersionFile})
			if err != nil {
				return fmt.Errorf("failed to initialize the stamper: %w", err)
			}

			stamppedVersion := stamper.Stamp(version)

			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			tmpBuildDir, err := ioutil.TempDir(wd, "build")
			if err != nil {
				return fmt.Errorf("failed to create temporary build directory: %w", err)
			}

			// Copy the original files to the build directory, resolving all symlinks.
			if err := filepath.Walk(packageDir, func(src string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("failed to process file %q: %w", src, err)
				}

				dst := filepath.Join(tmpBuildDir, src)
				resolved, err := filepath.EvalSymlinks(src)
				if err != nil {
					return fmt.Errorf("failed to resolve symlink for %q: %w", resolved, err)
				}
				if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
					return fmt.Errorf("failed to copy directory %q: %w", resolved, err)
				}
				if !info.IsDir() {
					if err := copyFile(dst, resolved); err != nil {
						return fmt.Errorf("failed to copy file %q: %w", resolved, err)
					}
				}

				return nil
			}); err != nil {
				return err
			}

			buildDir := filepath.Join(tmpBuildDir, packageDir)

			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			helmCmd := exec.CommandContext(
				ctx,
				helmPath, "package", buildDir,
				"--version", stamppedVersion,
				"--app-version", stamppedVersion,
			)
			var stdout strings.Builder
			helmCmd.Stdout = &stdout
			helmCmd.Stderr = os.Stderr
			if err := helmCmd.Run(); err != nil {
				return fmt.Errorf("failed to package chart: %w", err)
			}

			re := regexp.MustCompile("Successfully packaged chart and saved it to: (.*)")
			outputLocationMatch := re.FindStringSubmatch(stdout.String())

			if err := os.Rename(outputLocationMatch[1], output); err != nil {
				return fmt.Errorf("failed to move output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&helmPath, "helm-path", "", "The path to the Helm binary.")
	cmd.Flags().StringVar(&packageDir, "package-dir", "", "The Bazel package directory.")
	cmd.Flags().StringVar(&output, "output", "", "The output path for the .tgz file.")
	cmd.Flags().StringVar(&version, "version", "", "The chart version. It expands Bazel workspace statuses.")
	// https://github.com/bazelbuild/bazel/blob/a7e4a2b539aad8240672f19ac3c23b277f7915f8/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkRuleContextApi.java#L495-L502
	cmd.Flags().StringVar(&bazelInfoFile, "bazel-info-file", "", "The Bazel file that is used to hold the non-volatile workspace status for the current build request.")
	// https://github.com/bazelbuild/bazel/blob/a7e4a2b539aad8240672f19ac3c23b277f7915f8/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkRuleContextApi.java#L504-L511
	cmd.Flags().StringVar(&bazelVersionFile, "bazel-version-file", "", "The Bazel file that is used to hold the volatile workspace status for the current build request.")

	return cmd
}

func copyFile(dst, src string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
