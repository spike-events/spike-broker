module github.com/spike-events/spike-broker/test

go 1.23

toolchain go1.23.3

replace github.com/spike-events/spike-broker => ../

replace github.com/spike-events/spike-broker/v2 => ../v2/

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gofrs/uuid/v5 v5.3.0
	github.com/nats-io/nats.go v1.38.0
	github.com/spike-events/spike-broker v0.4.5
	github.com/spike-events/spike-broker/v2 v2.0.5
	github.com/stretchr/testify v1.9.0
	github.com/vincent-petithory/dataurl v1.0.0
	gorm.io/driver/sqlite v1.4.3
	gorm.io/gorm v1.25.12
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-chi/chi/v5 v5.2.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator v9.31.0+incompatible // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hetiansu5/urlquery v1.2.7 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.15 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.7.3 // indirect
	github.com/nats-io/nats-server/v2 v2.10.24 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
