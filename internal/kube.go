// Copyright 2017 CoreOS Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/coreos/container-linux-update-operator/pkg/k8sutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// installerEnvPath is the env file written by tectonic-installer
	installerEnvPath = "/etc/kubernetes/installer/kubelet.env"
	// envVersionKey is the key for the version flag
	envVersionKey = "KUBELET_IMAGE_TAG"
)

// WriteKubeletEnv writes the `kubelet.env` file
func (a *App) WriteKubeletEnv(destPath string, k8sVersion string) error {
	// This reverse charset constraints in docker tags (for the hyperkube image)
	kubeletVersion := strings.Replace(k8sVersion, "+", "_", -1)

	flags, err := readEnvFile(installerEnvPath)
	if err != nil {
		return nil
	}

	dstFp, err := os.Create(destPath)
	if err != nil {
		return nil
	}
	defer dstFp.Close()

	for k, v := range flags {
		if k == envVersionKey && kubeletVersion != "" {
			v = kubeletVersion
		}
		fmt.Fprintf(dstFp, "%s=%s\n", k, v)
	}

	return nil
}

// GetKubeVersion retrieves kubernetes version querying several sources:
//  1. a custom/forced version string
//  2. GitVersion of the remote API-server `/version`
//  3. hyperkube version (from its container tag)
func (a *App) GetKubeVersion() (string, error) {
	if a.Conf.ForceKubeVersion != "" {
		return a.Conf.ForceKubeVersion, nil
	}

	apiVersion, apiErr := a.versionFromAPIServer()
	if apiErr == nil {
		return apiVersion, nil
	}
	logrus.Warn("failed attempt to determine Kubernetes APIServer version: ", apiErr)

	pathVersion, pathErr := versionFromPath(installerEnvPath, envVersionKey)
	if pathErr == nil {
		logrus.Warn("Falling back to installer-provided kubernetes version")
		// This accomodates for charset constraints in docker tags (for the hyperkube image)
		version := strings.Replace(pathVersion, "_", "+", -1)
		return version, nil
	}
	logrus.Warn("failed attempt to determine Kubernetes installer version: ", pathErr)

	return "", errors.New("unable to determine cluster version")
}

// versionFromAPIServer connects to the APIServer and determines the kubernetes version
func (a *App) versionFromAPIServer() (string, error) {
	logrus.Info("Determining kubernetes version")
	config, err := clientcmd.BuildConfigFromFlags("", a.Conf.Kubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to build kubeconfig")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to build kube client")
	}

	version, err := client.ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get server version")
	}
	logrus.Debug("Got kubernetes version ", version.GitVersion)

	return version.GitVersion, nil
}

// versionFromPath reads Kubernetes version from a file
func versionFromPath(path string, envKey string) (string, error) {
	flags, err := readEnvFile(path)
	if err != nil {
		return "", err
	}

	version, ok := flags[envKey]
	if ok && version != "" {
		return version, nil
	}

	return "", errors.Errorf("no %q flag found in %q", envKey, path)
}

// readEnvFile reads a systemd env file and returns a map with
// the environment flags.
func readEnvFile(envPath string) (map[string]string, error) {
	env := make(map[string]string)

	fp, err := os.Open(envPath)
	if err != nil {
		return env, err
	}
	defer fp.Close()

	sc := bufio.NewScanner(fp)
	for sc.Scan() {
		line := sc.Text()
		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) == 2 {
			env[tokens[0]] = strings.Trim(tokens[1], `"`)
		}
	}

	return env, nil
}

// WriteNodeAnnotation writes the special annotation that indicates completion
// of the tool.
func (a *App) WriteNodeAnnotation() error {
	logrus.Infof("Writing node annotation %q", a.Conf.WriteNodeAnnotation)

	config, err := clientcmd.BuildConfigFromFlags("", a.Conf.Kubeconfig)
	if err != nil {
		return errors.Wrap(err, "failed to build kubeconfig")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to build kube client")
	}

	node := client.CoreV1().Nodes()

	annotations := map[string]string{
		a.Conf.WriteNodeAnnotation: "true",
	}

	err = k8sutil.SetNodeAnnotations(node, a.Conf.NodeName, annotations)
	if err != nil {
		return errors.Wrap(err, "unable to set node annotation")
	}

	return nil
}
