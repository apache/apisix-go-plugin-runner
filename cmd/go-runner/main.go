// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
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
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/apache/apisix-go-plugin-runner/internal/server"
)

var (
	InfoOut io.Writer = os.Stdout
)

func newVersionCommand() *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "version",
		Run: func(cmd *cobra.Command, _ []string) {
			if long {
				fmt.Fprint(InfoOut, longVersion())
			} else {
				fmt.Fprintf(InfoOut, "version %s\n", shortVersion())
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&long, "long", false, "show long mode version information")
	return cmd
}

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run",
		Run: func(cmd *cobra.Command, _ []string) {
			server.Run()
		},
	}
	return cmd
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apisix-go-plugin-runner [command]",
		Long:    "The Plugin runner to run Go plugins",
		Version: shortVersion(),
	}

	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newVersionCommand())
	return cmd
}

func main() {
	root := NewCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
