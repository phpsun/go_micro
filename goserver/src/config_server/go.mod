module config_server

go 1.15

require (
	common v1.0.0
	github.com/go-redis/redis v6.15.9+incompatible
	google.golang.org/protobuf v1.25.0 // indirect
	gorm.io/gorm v1.20.7
)

replace common => ../common
