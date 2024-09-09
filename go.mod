module github.com/speakeasy-api/sdk-gen-config

go 1.23

require (
	github.com/AlekSi/pointer v1.2.0
	github.com/daveshanley/vacuum v0.9.12
	github.com/google/uuid v1.5.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/stretchr/testify v1.9.0
	github.com/wk8/go-ordered-map/v2 v2.1.8
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/pb33f/doctor v0.0.4 // indirect
	github.com/pb33f/libopenapi v0.15.14 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	golang.org/x/exp v0.0.0-20240213143201-ec583247a57a // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

// Can be removed once https://github.com/wk8/go-ordered-map/pull/41 is merged and the dependency updated in libopenapi
replace github.com/wk8/go-ordered-map/v2 => github.com/speakeasy-api/go-ordered-map/v2 v2.0.0-20240813202817-2f1629387283
