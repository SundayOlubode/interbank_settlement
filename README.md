# interbank_settlement

A gross settlement system with atomic finality, designed for small-value, high-frequency retail transactions between Nigerian banks.

# Disclaimer

_All organizations mentioned are used strictly for academic and illustrative purposes. This project is not affiliated with or endorsed by any named institution._


interbank-settlement/            ← project root (git repo)
├── README.md
├── .env                         ← shared env vars (image tags, channel, ports…)
├── docker-compose.yaml
│
├── config/                      ← **source** configuration only
│   ├── configtx.yaml
│   ├── crypto-config.yaml
│   └── core.yaml                ← run-time defaults for peer/orderer CLIs
│
├── artifacts/                   ← **generated** at build/run time
│   ├── channel-artifacts/       ← genesis.block, *.tx, anchor-peer updates
│   └── organizations/           ← crypto material (cryptogen or Fabric-CA)
│       ├── ordererOrganizations/
│       │   └── cbn.naijachain.org/…
│       └── peerOrganizations/
│           ├── accessbank.naijachain.org/…
│           ├── gtbank.naijachain.org/…
│           ├── zenithbank.naijachain.org/…
│           └── firstbank.naijachain.org/…
│
├── chaincode/                   ← **source** for all chaincodes
│   └── account/
│       ├── go.mod
│       ├── main.go
│       └── metadata/
│           └── collections.json
│
├── cc-packages/                 ← *.tar.gz produced by `peer lifecycle package`
│   └── account_1.0.tar.gz
│
├── scripts/                     ← one-click helpers & CI hooks
│   ├── network.sh               ← up│down│createChannel│deployCC…
│   ├── utils.sh
│   ├── cli/                     ← run *inside* the CLI container
│   │   ├── install_cc.sh
│   │   ├── approve_org.sh
│   │   ├── commit_cc.sh
│   │   └── invoke_query.sh
│   └── cleanup.sh               ← rm -rf artifacts/, docker volume prune, logs
│
├── logs/                        ← peer & orderer logs copied out for debugging
│
└── docs/                        ← architecture diagrams, ADRs, HOWTOs
