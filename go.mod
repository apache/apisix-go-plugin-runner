module github.com/apache/apisix-go-plugin-runner

go 1.15

require (
	github.com/ReneKroon/ttlcache/v2 v2.4.0
	github.com/api7/ext-plugin-proto v0.6.0
	github.com/google/flatbuffers v2.0.0+incompatible
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/thediveo/enumflag v0.10.1
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/tools v0.1.9 // indirect
)

replace (
	github.com/miekg/dns v1.0.14 => github.com/miekg/dns v1.1.25
	// github.com/thediveo/enumflag@v0.10.1 depends on github.com/spf13/cobra@v0.0.7
	github.com/spf13/cobra v0.0.7 => github.com/spf13/cobra v1.2.1
)
