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

package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/tectonic-torcx/cli"
)

func main() {
	if err := cli.Init(); err != nil {
		logrus.Errorln(err)
		os.Exit(2)
	}

	if err := cli.MultiExecute(); err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}

	os.Exit(0)
}
