module github.com/Tangyd893/Scholar-Agent/agent-core

go 1.25.0

require (
	github.com/Tangyd893/Scholar-Agent v0.0.0
	github.com/Tangyd893/Scholar-Agent/proto/gen v0.0.0
)

require (
	github.com/openai/openai-go v1.12.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/Tangyd893/Scholar-Agent => ../
	github.com/Tangyd893/Scholar-Agent/proto/gen => ../proto/gen
)
