package transfer_test

import (
	"math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *TransferTestSuite) TestOnChanOpenInit() {
	var (
		channel      *channeltypes.Channel
		path         *ibctesting.Path
		chanCap      *capabilitytypes.Capability
		counterparty channeltypes.Counterparty
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"empty version string", func() {
				channel.Version = ""
			}, true,
		},
		{
			"max channels reached", func() {
				path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(math.MaxUint32 + 1)
			}, false,
		},
		{
			"invalid order - ORDERED", func() {
				channel.Ordering = channeltypes.ORDERED
			}, false,
		},
		{
			"invalid port ID", func() {
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockPort
			}, false,
		},
		{
			"invalid version", func() {
				channel.Version = "version" //nolint:goconst
			}, false,
		},
		{
			"capability already claimed", func() {
				err := suite.chainA.GetSimApp().ScopedTransferKeeper.ClaimCapability(suite.chainA.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty = channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        types.Version,
			}

			var err error
			chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			transferModule := transfer.NewIBCModule(suite.chainA.GetSimApp().TransferKeeper)
			version, err := transferModule.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, counterparty, channel.GetVersion(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(types.Version, version)
			} else {
				suite.Require().Error(err)
				suite.Require().Equal(version, "")
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanOpenTry() {
	var (
		channel             *channeltypes.Channel
		chanCap             *capabilitytypes.Capability
		path                *ibctesting.Path
		counterparty        channeltypes.Counterparty
		counterpartyVersion string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"max channels reached", func() {
				path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(math.MaxUint32 + 1)
			}, false,
		},
		{
			"capability already claimed", func() {
				err := suite.chainA.GetSimApp().ScopedTransferKeeper.ClaimCapability(suite.chainA.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}, false,
		},
		{
			"invalid order - ORDERED", func() {
				channel.Ordering = channeltypes.ORDERED
			}, false,
		},
		{
			"invalid port ID", func() {
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockPort
			}, false,
		},
		{
			"invalid counterparty version", func() {
				counterpartyVersion = "version"
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty = channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.TRYOPEN,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        types.Version,
			}
			counterpartyVersion = types.Version

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			tc.malleate() // explicitly change fields in channel and testChannel

			version, err := cbs.OnChanOpenTry(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, channel.Counterparty, counterpartyVersion,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(types.Version, version)
			} else {
				suite.Require().Error(err)
				suite.Require().Equal("", version)
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanOpenAck() {
	var counterpartyVersion string

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"invalid counterparty version", func() {
				counterpartyVersion = "version"
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			counterpartyVersion = types.Version

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = cbs.OnChanOpenAck(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointA.Counterparty.ChannelID, counterpartyVersion)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TransferTestSuite) TestPacketDataUnmarshalerInterface() {
	var (
		sender   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		receiver = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

		data          []byte
		expPacketData types.FungibleTokenPacketData
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: valid packet data with memo",
			func() {
				expPacketData = types.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "some memo",
				}
				data = expPacketData.GetBytes()
			},
			true,
		},
		{
			"success: valid packet data without memo",
			func() {
				expPacketData = types.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}
				data = expPacketData.GetBytes()
			},
			true,
		},
		{
			"failure: invalid packet data",
			func() {
				data = []byte("invalid packet data")
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		packetData, err := transfer.IBCModule{}.UnmarshalPacketData(data)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().Equal(expPacketData, packetData)
		} else {
			suite.Require().Error(err)
			suite.Require().Nil(packetData)
		}
	}
}
