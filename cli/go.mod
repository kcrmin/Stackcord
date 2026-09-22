module github.com/kcrmin/Stackcord/cli

go 1.26.0

require (
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	go.yaml.in/yaml/v3 v3.0.4
)

require (
	go.etcd.io/bbolt v1.5.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kcrmin/Stackcord/core v0.0.0
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/kcrmin/Stackcord/core => ../core
