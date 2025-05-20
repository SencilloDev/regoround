// Copyright 2025 Sencillo
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

package cmd

import (
	"runtime"
	"time"

	"github.com/SencilloDev/gupdate"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates regoround",
	RunE:  update,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func update(cmd *cobra.Command, args []string) error {
	gh := gupdate.GitHubProject{
		Name:         "regoround",
		Owner:        "SencilloDev",
		Platform:     runtime.GOOS,
		Arch:         runtime.GOARCH,
		ChecksumFunc: gupdate.GoReleaserChecksum,
	}

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
	s.Suffix = " updating regoroundctl..."
	s.Start()
	release, err := gupdate.GetLatestRelease(gh)
	if err != nil {
		return err
	}

	if err := release.Update(); err != nil {
		return err
	}
	s.Stop()

	return nil
}
