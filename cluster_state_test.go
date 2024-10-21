package ignite

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

type ClusterTestSuite struct {
	testing2.IgniteTestSuite
	client *Client
}

func TestClusterTestSuite(t *testing.T) {
	suite.Run(t, new(ClusterTestSuite))
}

func (suite *ClusterTestSuite) SetupSuite() {
	_, err := suite.StartIgnite(testing2.WithPersistence())
	if err != nil {
		suite.T().Fatal("failed to start ignite instance", err)
	}
	suite.client, err = StartTestClient(context.Background(), WithAddresses(defaultAddress+":10800"))
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
}

func (suite *ClusterTestSuite) TearDownSuite() {
	suite.KillAllGrids()
	if suite.client != nil {
		_ = suite.client.Close(context.Background())
		suite.client = nil
	}
}

func (suite *ClusterTestSuite) TestClusterState() {
	state, err := suite.client.GetClusterState(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), Inactive, state)

	err = suite.client.ChangeClusterState(context.Background(), Active)
	require.NoError(suite.T(), err)

	state, err = suite.client.GetClusterState(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), Active, state)

	err = suite.client.ChangeClusterState(context.Background(), ActiveReadOnly)
	require.NoError(suite.T(), err)

	state, err = suite.client.GetClusterState(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), ActiveReadOnly, state)

	err = suite.client.ChangeClusterState(context.Background(), Inactive)
	require.NoError(suite.T(), err)

	state, err = suite.client.GetClusterState(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), Inactive, state)
}

func (suite *ClusterTestSuite) TestWalState() {
	err := suite.client.ChangeClusterState(context.Background(), Active)
	require.NoError(suite.T(), err)

	_, err = suite.client.CreateCache(context.Background(), cacheName)
	require.NoError(suite.T(), err)

	isWalEnabled, err := suite.client.IsWalEnabled(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.True(suite.T(), isWalEnabled)

	isStateChanged, err := suite.client.DisableWal(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.True(suite.T(), isStateChanged)

	isStateChanged, err = suite.client.DisableWal(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.False(suite.T(), isStateChanged)

	isWalEnabled, err = suite.client.IsWalEnabled(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.False(suite.T(), isWalEnabled)

	isStateChanged, err = suite.client.EnableWal(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.True(suite.T(), isStateChanged)

	isStateChanged, err = suite.client.EnableWal(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.False(suite.T(), isStateChanged)

	isWalEnabled, err = suite.client.IsWalEnabled(context.Background(), cacheName)
	require.NoError(suite.T(), err)
	require.True(suite.T(), isWalEnabled)
}
