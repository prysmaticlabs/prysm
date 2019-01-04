module github.com/prysmaticlabs/prysm

require (
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/allegro/bigcache v1.1.0 // indirect
	github.com/aristanetworks/goarista v0.0.0-20190103214300-332ede26fbf0
	github.com/boltdb/bolt v1.3.1
	github.com/btcsuite/btcd v0.0.0-20181130015935-7d2daa5bfef2 // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.8.20
	github.com/fd/go-nat v1.0.0 // indirect
	github.com/fjl/memsize v0.0.0-20180929194037-2a09253e352a
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/gogo/protobuf v1.2.0
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.2.0
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/gxed/hashland v0.0.0-20180221191214-d9f6b97f8db2 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/huin/goupnp v1.0.0 // indirect
	github.com/ipfs/go-cid v0.9.0 // indirect
	github.com/ipfs/go-datastore v3.2.0+incompatible
	github.com/ipfs/go-ipfs-addr v0.1.25
	github.com/ipfs/go-ipfs-util v1.2.8 // indirect
	github.com/ipfs/go-log v1.5.7
	github.com/ipfs/go-todocounter v1.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2 // indirect
	github.com/jbenet/goprocess v0.0.0-20160826012719-b497e2f366b8 // indirect
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/karalabe/hid v0.0.0-20181128192157-d815e0c1a2e2 // indirect
	github.com/libp2p/go-addr-util v2.0.7+incompatible // indirect
	github.com/libp2p/go-buffer-pool v0.1.1 // indirect
	github.com/libp2p/go-conn-security v0.1.15 // indirect
	github.com/libp2p/go-conn-security-multistream v0.1.15 // indirect
	github.com/libp2p/go-flow-metrics v0.2.0 // indirect
	github.com/libp2p/go-libp2p v6.0.29+incompatible
	github.com/libp2p/go-libp2p-autonat v0.0.0-20181221182007-93b1787f76de // indirect
	github.com/libp2p/go-libp2p-blankhost v0.3.15
	github.com/libp2p/go-libp2p-circuit v2.3.2+incompatible
	github.com/libp2p/go-libp2p-crypto v2.0.1+incompatible
	github.com/libp2p/go-libp2p-discovery v0.0.0-20181218190140-cc4105e21706 // indirect
	github.com/libp2p/go-libp2p-host v3.0.15+incompatible
	github.com/libp2p/go-libp2p-interface-connmgr v0.0.21 // indirect
	github.com/libp2p/go-libp2p-interface-pnet v3.0.0+incompatible // indirect
	github.com/libp2p/go-libp2p-kad-dht v4.4.12+incompatible
	github.com/libp2p/go-libp2p-kbucket v2.2.12+incompatible // indirect
	github.com/libp2p/go-libp2p-loggables v1.1.24 // indirect
	github.com/libp2p/go-libp2p-metrics v2.1.7+incompatible // indirect
	github.com/libp2p/go-libp2p-nat v0.8.8 // indirect
	github.com/libp2p/go-libp2p-net v3.0.15+incompatible
	github.com/libp2p/go-libp2p-peer v2.4.0+incompatible // indirect
	github.com/libp2p/go-libp2p-peerstore v2.0.6+incompatible
	github.com/libp2p/go-libp2p-protocol v1.0.0 // indirect
	github.com/libp2p/go-libp2p-pubsub v0.10.2
	github.com/libp2p/go-libp2p-record v4.1.7+incompatible // indirect
	github.com/libp2p/go-libp2p-routing v2.7.1+incompatible // indirect
	github.com/libp2p/go-libp2p-secio v2.0.17+incompatible // indirect
	github.com/libp2p/go-libp2p-swarm v3.0.22+incompatible
	github.com/libp2p/go-libp2p-transport v3.0.15+incompatible // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.16 // indirect
	github.com/libp2p/go-maddr-filter v1.1.10 // indirect
	github.com/libp2p/go-mplex v0.2.30 // indirect
	github.com/libp2p/go-msgio v0.0.6 // indirect
	github.com/libp2p/go-reuseport v0.1.18 // indirect
	github.com/libp2p/go-reuseport-transport v0.1.11 // indirect
	github.com/libp2p/go-sockaddr v1.0.3 // indirect
	github.com/libp2p/go-stream-muxer v3.0.1+incompatible // indirect
	github.com/libp2p/go-tcp-transport v2.0.16+incompatible // indirect
	github.com/libp2p/go-testutil v1.2.10 // indirect
	github.com/libp2p/go-ws-transport v2.0.15+incompatible // indirect
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/miekg/dns v1.1.1 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v0.0.0-20181005183134-51976451ce19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mr-tron/base58 v1.1.0 // indirect
	github.com/multiformats/go-multiaddr v1.4.0
	github.com/multiformats/go-multiaddr-dns v0.2.5 // indirect
	github.com/multiformats/go-multiaddr-net v1.7.1 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multihash v1.0.8 // indirect
	github.com/multiformats/go-multistream v0.3.9 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.0 // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/prometheus v2.1.0+incompatible // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/rs/cors v1.6.0 // indirect
	github.com/sirupsen/logrus v1.2.0
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 // indirect
	github.com/steakknife/hamming v0.0.0-20180906055917-c99c65617cd3
	github.com/syndtr/goleveldb v0.0.0-20181128100959-b001fa50d6b2
	github.com/urfave/cli v1.20.0
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc // indirect
	github.com/whyrusleeping/go-notifier v0.0.0-20170827234753-097c5d47330f // indirect
	github.com/whyrusleeping/go-smux-multiplex v3.0.16+incompatible // indirect
	github.com/whyrusleeping/go-smux-multistream v2.0.2+incompatible // indirect
	github.com/whyrusleeping/go-smux-yamux v2.0.8+incompatible // indirect
	github.com/whyrusleeping/mafmt v1.2.8 // indirect
	github.com/whyrusleeping/mdns v0.0.0-20180901202407-ef14215e6b30 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	github.com/whyrusleeping/yamux v1.1.2 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	go.opencensus.io v0.18.0
	golang.org/x/crypto v0.0.0-20190103213133-ff983b9c42bc
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3
	golang.org/x/sys v0.0.0-20190102155601-82a175fd1598 // indirect
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	google.golang.org/grpc v1.17.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0 // indirect
	k8s.io/api v0.0.0-20181221193117-173ce66c1e39 // indirect
	k8s.io/apimachinery v0.0.0-20190103171832-0cd393178e09
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v0.1.0 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
