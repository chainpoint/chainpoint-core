module github.com/chainpoint/chainpoint-core

go 1.16

require (
	github.com/btcsuite/btcd v0.22.0-beta.0.20211005184431-e3449998be39
	github.com/btcsuite/btcutil v1.0.3-0.20210527170813-e2ba6805a890
	github.com/chainpoint/leader-election v0.0.0
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/drand/drand v1.0.0-rc1
	github.com/enriquebris/goconcurrentqueue v0.6.0
	github.com/ethereum/go-ethereum v1.9.15
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-redis/redis v6.15.8+incompatible
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/jacohend/flag v1.10.1-0.20210910180111-f81aa67342a2
	github.com/knq/pemutil v0.0.0-20181215144041-fb6fad722528
	github.com/lestrrat-go/jwx v0.9.2
	github.com/lightningnetwork/lnd v0.9.2-beta
	github.com/ltcsuite/ltcd v0.20.1-beta // indirect
	github.com/manifoldco/promptui v0.8.0
	github.com/mitchellh/mapstructure v1.3.2 // indirect
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/sethvargo/go-password v0.2.0
	github.com/spf13/afero v1.3.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/go-amino v0.15.1 // indirect
	github.com/tendermint/tendermint v0.33.5-0.20200528083845-9ee3e4896bf8
	github.com/tendermint/tm-db v0.5.1
	github.com/throttled/throttled/v2 v2.8.0
	google.golang.org/grpc v1.38.0
	gopkg.in/ini.v1 v1.57.0 // indirect
	gopkg.in/macaroon-bakery.v2 v2.2.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
)

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5
	github.com/lightningnetwork/lnd v0.9.2-beta => github.com/tierion/lnd v0.9.0-beta-rc1.0.20220119230648-75c2b68027ff
	github.com/lightningnetwork/lnd/lnrpc/invoicesrpc v0.9.2-beta => github.com/tierion/lnd/lnrpc/invoicesrpc v0.9.0-beta-rc1.0.20220119230648-75c2b68027ff
	github.com/lightningnetwork/lnd/lnrpc/signrpc v0.9.2-beta => github.com/tierion/lnd/lnrpc/signrpc v0.9.0-beta-rc1.0.20220119230648-75c2b68027ff
	github.com/lightningnetwork/lnd/lnrpc/walletrpc v0.9.2-beta => github.com/tierion/lnd/lnrpc/walletrpc v0.9.0-beta-rc1.0.20220119230648-75c2b68027ff
	github.com/lightningnetwork/lnd/lntypes v0.9.2-beta => github.com/tierion/lnd/lntypes v0.9.0-beta-rc1.0.20220119230648-75c2b68027ff
	github.com/lightningnetwork/lnd/macaroons v0.9.2-beta => github.com/tierion/lnd/macaroons v0.9.0-beta-rc1.0.20220119230648-75c2b68027ff
	go.uber.org/atomic => github.com/uber-go/atomic v1.5.0
)
