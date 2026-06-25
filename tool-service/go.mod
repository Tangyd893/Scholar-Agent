module github.com/Tangyd893/Scholar-Agent/tool-service

go 1.25.0

require (
	github.com/Tangyd893/Scholar-Agent v0.0.0
	github.com/Tangyd893/Scholar-Agent/proto/gen v0.0.0
)

require (
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
