module github.com/chainpoint/chainpoint-core/go-abci-service

go 1.16

require (
	github.com/btcsuite/btcd v0.21.0-beta.0.20201208033208-6bd4c64a54fa
	github.com/btcsuite/btcutil v1.0.2
	github.com/drand/drand v1.0.0-rc1
	github.com/enriquebris/goconcurrentqueue v0.6.0
	github.com/ethereum/go-ethereum v1.9.15
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-redis/redis v6.15.8+incompatible
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/knq/pemutil v0.0.0-20181215144041-fb6fad722528
	github.com/lestrrat-go/jwx v0.9.2
	github.com/lib/pq v1.7.0 // indirect
	github.com/lightningnetwork/lnd v0.9.2-beta
	github.com/ltcsuite/ltcd v0.20.1-beta // indirect
	github.com/miekg/dns v1.1.29 // indirect
	github.com/mitchellh/mapstructure v1.3.2 // indirect
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/robert-zaremba/flag v1.10.1
	github.com/spf13/afero v1.3.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/go-amino v0.15.1 // indirect
	github.com/tendermint/tendermint v0.33.5-0.20200528083845-9ee3e4896bf8
	github.com/tendermint/tm-db v0.5.1
	github.com/throttled/throttled/v2 v2.8.0
	google.golang.org/genproto v0.0.0-20200619004808-3e7fca5c55db // indirect
	google.golang.org/grpc v1.29.1
	gopkg.in/ini.v1 v1.57.0 // indirect
	gopkg.in/macaroon-bakery.v2 v2.2.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5
	github.com/lightninglabs/neutrino => github.com/Tierion/neutrino v0.11.1-0.20210510140741-bcbc10e7e12e
	github.com/lightningnetwork/lnd v0.9.2-beta => github.com/tierion/lnd v0.9.0-beta-rc1.0.20210513144118-84217725bb47
	github.com/lightningnetwork/lnd/lnrpc/invoicesrpc v0.9.2-beta => github.com/tierion/lnd/lnrpc/invoicesrpc v0.9.0-beta-rc1.0.20210513144118-84217725bb47
	github.com/lightningnetwork/lnd/lnrpc/signrpc v0.9.2-beta => github.com/tierion/lnd/lnrpc/signrpc v0.9.0-beta-rc1.0.20210513144118-84217725bb47
	github.com/lightningnetwork/lnd/lnrpc/walletrpc v0.9.2-beta => github.com/tierion/lnd/lnrpc/walletrpc v0.9.0-beta-rc1.0.20210513144118-84217725bb47
	github.com/lightningnetwork/lnd/lntypes v0.9.2-beta => github.com/tierion/lnd/lntypes v0.9.0-beta-rc1.0.20210513144118-84217725bb47
	github.com/lightningnetwork/lnd/macaroons v0.9.2-beta => github.com/tierion/lnd/macaroons v0.9.0-beta-rc1.0.20210513144118-84217725bb47
	go.uber.org/atomic => github.com/uber-go/atomic v1.5.0
)
