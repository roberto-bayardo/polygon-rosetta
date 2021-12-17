// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configuration

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/maticnetwork/polygon-rosetta/polygon"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/params"
)

// Mode is the setting that determines if
// the implementation is "online" or "offline".
type Mode string

const (
	// Online is when the implementation is permitted
	// to make outbound connections.
	Online Mode = "ONLINE"

	// Offline is when the implementation is not permitted
	// to make outbound connections.
	Offline Mode = "OFFLINE"

	// Mainnet is the Polygon Mainnet.
	Mainnet string = "MAINNET"

	// Mumbai is the Polygon testnet
	Mumbai string = "MUMBAI"

	// Testnet defaults to `Mumbai`
	Testnet string = "TESTNET"

	// DataDirectory is the default location for all
	// persistent data.
	DataDirectory = "/data"

	// ModeEnv is the environment variable read
	// to determine mode.
	ModeEnv = "MODE"

	// NetworkEnv is the environment variable
	// read to determine network.
	NetworkEnv = "NETWORK"

	// PortEnv is the environment variable
	// read to determine the port for the Rosetta
	// implementation.
	PortEnv = "PORT"

	// BorEnv is an optional environment variable
	// used to connect polygon-rosetta to an already
	// running bor node.
	BorEnv = "BOR"

	// DefaultBorURL is the default URL for
	// a running geth node. This is used
	// when GethEnv is not populated.
	DefaultBorURL = "http://localhost:8545"

	// SkipGethAdminEnv is an optional environment variable
	// to skip geth `admin` calls which are typically not supported
	// by hosted node services. When not set, defaults to false.
	SkipGethAdminEnv = "SKIP_GETH_ADMIN"

	// LeanTracesEnv is an optional environment variable which
	// trims raw block trace and receipt data from block responses.
	// Defaults to false.
	LeanTracesEnv = "LEAN_TRACES"

	// GethHeadersEnv is an optional environment variable
	// of a comma-separated list of key:value pairs to apply
	// to geth clients as headers. When not set, defaults to []
	GethHeadersEnv = "GETH_HEADERS"

	// MiddlewareVersion is the version of polygon-rosetta.
	MiddlewareVersion = "0.0.4"

	// MainnetChainID is the chain ID for Mainnet
	MainnetChainID = "137"

	// MumbaiChainID is the chain ID for Mumbai Testnet
	MumbaiChainID = "80001"
)

// Configuration determines how
type Configuration struct {
	Mode                   Mode
	Network                *types.NetworkIdentifier
	GenesisBlockIdentifier *types.BlockIdentifier
	BorURL                 string
	RemoteGeth             bool
	Port                   int
	SkipGethAdmin          bool
	GethHeaders            []*polygon.HTTPHeader
	LeanTraces             bool

	// Block Reward Data
	Params *params.ChainConfig

	// Governance contract where the token will be sent to and burnt in london fork
	BurntContract map[string]string
}

// LoadConfiguration attempts to create a new Configuration
// using the ENVs in the environment.
func LoadConfiguration() (*Configuration, error) {
	config := &Configuration{}

	modeValue := Mode(os.Getenv(ModeEnv))
	switch modeValue {
	case Online:
		config.Mode = Online
	case Offline:
		config.Mode = Offline
	case "":
		return nil, errors.New("MODE must be populated")
	default:
		return nil, fmt.Errorf("%s is not a valid mode", modeValue)
	}

	networkValue := os.Getenv(NetworkEnv)
	switch networkValue {
	case Mainnet:
		config.Network = &types.NetworkIdentifier{
			Blockchain: polygon.Blockchain,
			Network:    polygon.MainnetNetwork,
		}
		config.GenesisBlockIdentifier = polygon.MainnetGenesisBlockIdentifier
		config.Params = params.MainnetChainConfig
		config.Params.ChainID.SetString(MainnetChainID, 10)
		config.BurntContract = map[string]string{
			"0": "0x0000000000000000000000000000000000000000",
		} // TODO add burn contract and block when mainnet forks to support EIP-1559
	case Testnet, Mumbai:
		config.Network = &types.NetworkIdentifier{
			Blockchain: polygon.Blockchain,
			Network:    polygon.TestnetNetwork,
		}
		config.GenesisBlockIdentifier = polygon.MumbaiGenesisBlockIdentifier
		config.Params = params.GoerliChainConfig
		config.Params.ChainID.SetString(MumbaiChainID, 10)
		config.BurntContract = map[string]string{
			"22640000": "0x70bcA57F4579f58670aB2d18Ef16e02C17553C38",
		}
	case "":
		return nil, errors.New("NETWORK must be populated")
	default:
		return nil, fmt.Errorf("%s is not a valid network", networkValue)
	}

	config.BorURL = DefaultBorURL
	envBorURL := os.Getenv(BorEnv)
	if len(envBorURL) > 0 {
		config.RemoteGeth = true
		config.BorURL = envBorURL
	}

	config.SkipGethAdmin = false
	envSkipGethAdmin := os.Getenv(SkipGethAdminEnv)
	if len(envSkipGethAdmin) > 0 {
		val, err := strconv.ParseBool(envSkipGethAdmin)
		if err != nil {
			return nil, fmt.Errorf("%w: unable to parse SKIP_GETH_ADMIN %s", err, envSkipGethAdmin)
		}
		config.SkipGethAdmin = val
	}

	config.LeanTraces = false
	envLeanTraces := os.Getenv(LeanTracesEnv)
	if len(envLeanTraces) > 0 {
		val, err := strconv.ParseBool(envLeanTraces)
		if err != nil {
			return nil, fmt.Errorf("%w: unable to parse LEAN_TRACES %s", err, envLeanTraces)
		}
		config.LeanTraces = val
	}

	envGethHeaders := os.Getenv(GethHeadersEnv)
	if len(envGethHeaders) > 0 {
		headers := strings.Split(envGethHeaders, ",")
		headerKVs := make([]*polygon.HTTPHeader, len(headers))
		for i, pair := range headers {
			kv := strings.Split(pair, ":")
			headerKVs[i] = &polygon.HTTPHeader{
				Key:   kv[0],
				Value: kv[1],
			}
		}
		config.GethHeaders = headerKVs
	}

	portValue := os.Getenv(PortEnv)
	if len(portValue) == 0 {
		return nil, errors.New("PORT must be populated")
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || len(portValue) == 0 || port <= 0 {
		return nil, fmt.Errorf("%w: unable to parse port %s", err, portValue)
	}
	config.Port = port

	return config, nil
}
