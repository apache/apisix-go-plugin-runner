<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

# Go Plugin Runner for Apache APISIX

[![Go Report Card](https://goreportcard.com/badge/github.com/apache/apisix-go-plugin-runner)](https://goreportcard.com/report/github.com/apache/apisix-go-plugin-runner)
[![Build Status](https://github.com/apache/apisix-go-plugin-runner/workflows/unit-test-ci/badge.svg?branch=master)](https://github.com/apache/apisix-go-plugin-runner/actions)
[![Codecov](https://codecov.io/gh/apache/apisix-go-plugin-runner/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/apisix-go-plugin-runner)
[![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/apache/apisix-go-plugin-runner)

Runs [Apache APISIX](http://apisix.apache.org/) plugins written in Go. Implemented as a sidecar that accompanies APISIX.

## Status

This project is currently considered experimental.

## Why apisix-go-plugin-runner

Apache APISIX offers many full-featured plugins covering areas such as authentication, security, traffic control, serverless, analytics & monitoring, transformations, logging.

It also provides highly extensible API, allowing common phases to be mounted, and users can use these API to develop their own plugins.

This project is APISIX Go side implementation that supports writing plugins in Go.

Currently, Go Plugin Runner is provided as a library. This is because the convention of Go is to compile all the code into an executable file. 

Although there is a mechanism for Go Plugin to compile the plugin code into a dynamic link library and then load it into the binary. But as far as experience is concerned, there are still some imperfections that are not so simple and direct to use.

The structure of the apache/apisix-go-plugin-runner repository on GitHub is as follows:

```
.
├── cmd
├── internal
├── pkg
```

`internal` is responsible for the internal implementation, `pkg` displays the external interface, and `cmd` provides examples of the demonstration.
There is a subdirectory of `go-runner` under the `cmd` directory. By reading the code in this section, you can learn how to use Go Plugin Runner in practical applications.

## How it Works

At present, the communication between Go Plugin Runner and Apache APISIX is an RPC based on Unix socket. So Go Plugin Runner and Apache APISIX need to be deployed on the same machine.

### Enable Go Plugin Runner

As mentioned earlier, Go Plugin Runner is managed by Apache APISIX, which runs as a child process of APISIX. So we have to configure and run this Runner in Apache APISIX.

The following configuration process will take the code `cmd/go-runner` in the `apisix-go-plugin-runner` project as an example.

1. Compile the sample code. Executing `make build` generates the executable file go-runner.
2. Make the following configuration in the conf/config.yaml file of Apache APISIX:

```yaml
ext-plugin:
  cmd: ["/path/to/apisix-go-plugin-runner/go-runner", "run"]
```

With the above configuration, Apache APISIX pulls up `go-runner` when it starts and closes `go-runner` when it stops.

In view of the fact that `apisix-go-plugin-runner` is used in the form of a library in the actual development process, you need to replace the above example configuration with your own executable and startup instructions.

Finally, after the startup of Apache APISIX, `go-runner` will be started along with it.

### Other configuration methods

Of course, if you need to take these three steps every time you verify the functionality in the development process, it is quite tedious. So we also provide another configuration that allows apisix-go-plugin-runner to run independently during development.

1. The first thing to do is to compile the code.
2. Configure the following in the conf/config.yaml file of Apache APISIX:

```yaml
ext-plugin:
  path_for_test: /tmp/runner.sock
```

3. Start `go-runner` with the following code.

```
APISIX_LISTEN_ADDRESS=unix:/tmp/runner.sock ./go-runner run
```

Notice that we specify the socket address to be used for `go-runner` communication through the environment variable `APISIX_LISTEN_ADDRESS`. This address needs to be consistent with the configuration in Apache APISIX.

## License

Apache 2.0 LICENSE
