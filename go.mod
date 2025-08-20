module github.com/aws/aws-xray-daemon

go 1.23.0

toolchain go1.24.1

require (
	github.com/aws/aws-sdk-go-v2 v1.38.0
	github.com/aws/aws-sdk-go-v2/config v1.31.1
	github.com/aws/aws-sdk-go-v2/credentials v1.18.5
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.37.1
	github.com/aws/aws-sdk-go-v2/service/xray v1.34.1
	github.com/aws/smithy-go v1.22.5
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575
	github.com/shirou/gopsutil v2.19.10+incompatible
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.38.0
	golang.org/x/sys v0.31.0
	gopkg.in/yaml.v2 v2.2.8
)

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.28.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.33.1 // indirect
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.1.0 // indirect
	golang.org/x/text v0.23.0 // indirect
)
