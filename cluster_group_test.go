package ignite

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"
)

const igniteNameAttribute = "org.apache.ignite.ignite.name"
const igniteConsistentIdAttribute = "org.apache.ignite.consistent.id"

type ClusterGroupTestSuite struct {
	testing2.IgniteTestSuite
	client *Client
}

var serverStartParameters = []func(params *testing2.IgniteParams){
	testing2.WithTcpDiscoverySpi("ru.gitverse.sbertech.ignite.discovery.TestLocalHostNameTcpDiscoverySpi"),
}

func TestClusterGroupTestSuite(t *testing.T) {
	suite.Run(t, new(ClusterGroupTestSuite))
}

func (suite *ClusterGroupTestSuite) SetupSuite() {
	_, err := suite.StartIgnite(serverStartParameters...)
	if err != nil {
		suite.T().Fatal("failed to start ignite instance", err)
	}
	_, err = suite.StartIgnite(serverStartParameters...)
	if err != nil {
		suite.T().Fatal("failed to start ignite instance", err)
	}
	_, err = suite.StartIgnite(testing2.WithClientMode())
	if err != nil {
		suite.T().Fatal("failed to start ignite instance", err)
	}
	suite.client, err = StartTestClient(context.Background(), WithAddresses(defaultAddress+":10800"))
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
}

func (suite *ClusterGroupTestSuite) TearDownSuite() {
	suite.KillAllGrids()
	if suite.client != nil {
		_ = suite.client.Close(context.Background())
		suite.client = nil
	}
}

func (suite *ClusterGroupTestSuite) TestClusterGroupNodes() {
	group, err := suite.client.GetClusterGroup()
	require.NoError(suite.T(), err)
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), 3, len(nodes))
	nodesByName := suite.toNodesByName(nodes)

	node, ok := nodesByName["srv_0"]
	require.True(suite.T(), ok)
	suite.checkNodeParameters(node, 0, false, true, 1)
	node, ok = nodesByName["srv_1"]
	require.True(suite.T(), ok)
	suite.checkNodeParameters(node, 1, false, false, 2)
	node, ok = nodesByName["srv_2"]
	require.True(suite.T(), ok)
	suite.checkNodeParameters(node, 2, true, false, 3)
}

func (suite *ClusterGroupTestSuite) TestServerFilter() {
	group, err := suite.client.GetClusterGroup(ForServers())
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_0", "srv_1")
}

func (suite *ClusterGroupTestSuite) TestClientFilter() {
	group, err := suite.client.GetClusterGroup(ForClients())
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_2")
}

func (suite *ClusterGroupTestSuite) TestNodeIdsFilter() {
	allNodes := suite.requestAllNodes()
	srvNode, ok := allNodes["srv_1"]
	require.True(suite.T(), ok)
	cliNode, ok := allNodes["srv_2"]
	require.True(suite.T(), ok)

	group, err := suite.client.GetClusterGroup(ForNodeIds(srvNode.Id(), cliNode.Id()), ForNodeIds(cliNode.Id(), srvNode.Id(), allNodes["srv_0"].Id()))
	require.NoError(suite.T(), err)

	nodes, err := group.Nodes(context.Background())
	nodesByName := suite.toNodesByName(nodes)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, len(nodes))
	require.True(suite.T(), reflect.DeepEqual(srvNode, nodesByName["srv_1"]))
	require.True(suite.T(), reflect.DeepEqual(cliNode, nodesByName["srv_2"]))

	node, err := group.Node(context.Background(), allNodes["srv_0"].id)
	require.NoError(suite.T(), err)
	require.Nil(suite.T(), node)

	node, err = group.Node(context.Background(), allNodes["srv_1"].id)
	require.NoError(suite.T(), err)
	require.True(suite.T(), reflect.DeepEqual(node, nodesByName["srv_1"]))

	node, err = group.Node(context.Background(), allNodes["srv_2"].id)
	require.NoError(suite.T(), err)
	require.True(suite.T(), reflect.DeepEqual(node, nodesByName["srv_2"]))
}

func (suite *ClusterGroupTestSuite) TestRandomFilter() {
	group, err := suite.client.GetClusterGroup(ForRandom())
	require.NoError(suite.T(), err)
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(nodes))
}

func (suite *ClusterGroupTestSuite) TestYoungestFilter() {
	group, err := suite.client.GetClusterGroup(ForYoungest())
	require.NoError(suite.T(), err)
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(nodes))
	require.Equal(suite.T(), "srv_2", nodes[0].ConsistentId())
}

func (suite *ClusterGroupTestSuite) TestOldestFilter() {
	group, err := suite.client.GetClusterGroup(ForOldest())
	require.NoError(suite.T(), err)
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(nodes))
	require.Equal(suite.T(), "srv_0", nodes[0].ConsistentId())
}

func (suite *ClusterGroupTestSuite) TestPredicateFilter() {
	group, err := suite.client.GetClusterGroup(ForPredicate(func(node *ClusterNode) bool {
		return node.Order() == 2
	}))
	require.NoError(suite.T(), err)
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, len(nodes))
	require.Equal(suite.T(), "srv_1", nodes[0].ConsistentId())
}

func (suite *ClusterGroupTestSuite) TestClusterNodesFilter() {
	allNodes := suite.requestAllNodes()
	srvNode, ok := allNodes["srv_1"]
	require.True(suite.T(), ok)
	cliNode, ok := allNodes["srv_2"]
	require.True(suite.T(), ok)

	_, err := suite.client.GetClusterGroup(ForClusterNodes(nil))
	require.Error(suite.T(), err)

	group, err := suite.client.GetClusterGroup(ForClusterNodes(srvNode, cliNode))
	require.NoError(suite.T(), err)

	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, len(nodes))
	nodesByName := suite.toNodesByName(nodes)
	require.True(suite.T(), reflect.DeepEqual(srvNode, nodesByName["srv_1"]))
	require.True(suite.T(), reflect.DeepEqual(cliNode, nodesByName["srv_2"]))
}

func (suite *ClusterGroupTestSuite) TestExceptNodesFilter() {
	_, err := suite.client.GetClusterGroup(ExceptNodes(nil))
	require.Error(suite.T(), err)

	allNodes := suite.requestAllNodes()
	group, err := suite.client.GetClusterGroup(ExceptNodes(allNodes["srv_1"], allNodes["srv_2"]))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_0")
}

func (suite *ClusterGroupTestSuite) TestExceptGroupNodesFilter() {
	_, err := suite.client.GetClusterGroup(ExceptGroupNodes(nil))
	require.Error(suite.T(), err)

	filterGroup, err := suite.client.GetClusterGroup(ForClients())
	require.NoError(suite.T(), err)

	group, err := suite.client.GetClusterGroup(ExceptGroupNodes(filterGroup))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_0", "srv_1")
}

func (suite *ClusterGroupTestSuite) TestNodeHostFilter() {
	_, err := suite.client.GetClusterGroup(ForHost(nil))
	require.Error(suite.T(), err)

	allNodes := suite.requestAllNodes()
	group, err := suite.client.GetClusterGroup(ForHost(allNodes["srv_1"]))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_0", "srv_1", "srv_2")
}

func (suite *ClusterGroupTestSuite) TestHostFilter() {
	group, err := suite.client.GetClusterGroup(ForHostNames("localhost"))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_0", "srv_1")
}

func (suite *ClusterGroupTestSuite) TestEmptyFilterResult() {
	group, err := suite.client.GetClusterGroup(ForServers(), ForClients())
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group)

	group, err = suite.client.GetClusterGroup(ForAttribute(igniteNameAttribute, "srv_0"), ForAttribute(igniteNameAttribute, "srv_1"))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group)

	group, err = suite.client.GetClusterGroup(ForNodeIds())
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group)

	allNodes := suite.requestAllNodes()
	group, err = suite.client.GetClusterGroup(ForNodeIds(allNodes["srv_1"].Id()), ForNodeIds(allNodes["srv_2"].Id()))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group)
}

func (suite *ClusterGroupTestSuite) TestMultipleServerSideFilters() {
	group, err := suite.client.GetClusterGroup(ForAttribute(igniteNameAttribute, "srv_0"), ForServers(), ForAttribute(igniteConsistentIdAttribute, "srv_0"))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(group, "srv_0")
}

func (suite *ClusterGroupTestSuite) TestClientFiltersReordering() {
	group, err := suite.client.GetClusterGroup(ForYoungest(), ForServers())
	require.NoError(suite.T(), err)
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), 0, len(nodes))
}

func (suite *ClusterGroupTestSuite) TestGroupNodeById() {
	allNodes := suite.requestAllNodes()
	cliNode := allNodes["srv_2"]

	group, err := suite.client.GetClusterGroup(ForClients())
	require.NoError(suite.T(), err)

	node, err := group.Node(context.Background(), cliNode.Id())
	require.NoError(suite.T(), err)
	require.True(suite.T(), reflect.DeepEqual(cliNode, node))

	node, err = group.Node(context.Background(), allNodes["srv_0"].Id())
	require.NoError(suite.T(), err)
	require.Nil(suite.T(), node)
}

func (suite *ClusterGroupTestSuite) TestTopologyUpdate() {
	allNodesGroup, err := suite.client.GetClusterGroup()
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(allNodesGroup, "srv_0", "srv_1", "srv_2")

	_, err = suite.StartIgnite(serverStartParameters...)
	require.NoError(suite.T(), err)
	defer func() {
		_ = suite.KillIgnite(3)
	}()

	nodes := suite.assertNodesInGroup(allNodesGroup, "srv_0", "srv_1", "srv_2", "srv_3")

	var nodeIds []uuid.UUID
	for _, node := range nodes {
		nodeIds = append(nodeIds, node.id)
	}
	nodeIdsGroup, err := suite.client.GetClusterGroup(ForNodeIds(nodeIds...))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(nodeIdsGroup, "srv_0", "srv_1", "srv_2", "srv_3")

	err = suite.KillIgnite(3)
	require.NoError(suite.T(), err)
	require.True(suite.T(), suite.WaitForTopologyVersion(5, 10*time.Second))

	srv3Node := nodes["srv_3"]

	suite.assertNodesInGroup(allNodesGroup, "srv_0", "srv_1", "srv_2")
	node, err := allNodesGroup.Node(context.Background(), srv3Node.Id())
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), node)

	suite.assertNodesInGroup(nodeIdsGroup, "srv_0", "srv_1", "srv_2")
	node, err = nodeIdsGroup.Node(context.Background(), srv3Node.Id())
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), node)

	srv3NodeGroup, err := suite.client.GetClusterGroup(ForClusterNodes(srv3Node))
	require.NoError(suite.T(), err)
	suite.assertNodesInGroup(srv3NodeGroup)
	node, err = srv3NodeGroup.Node(context.Background(), srv3Node.Id())
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), node)
}

func (suite *ClusterGroupTestSuite) assertNodesInGroup(group *ClusterGroup, names ...string) (nodesByName map[string]*ClusterNode) {
	nodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), len(names), len(nodes))
	nodesByName = suite.toNodesByName(nodes)
	for _, name := range names {
		_, ok := nodesByName[name]
		require.True(suite.T(), ok)
	}
	return
}

func (suite *ClusterGroupTestSuite) requestAllNodes() map[string]*ClusterNode {
	group, err := suite.client.GetClusterGroup()
	require.NoError(suite.T(), err)
	clusterNodes, err := group.Nodes(context.Background())
	require.NoError(suite.T(), err)
	return suite.toNodesByName(clusterNodes)
}

func (suite *ClusterGroupTestSuite) toNodesByName(nodes []*ClusterNode) map[string]*ClusterNode {
	res := make(map[string]*ClusterNode)
	for _, node := range nodes {
		attr, ok := node.Attributes()[igniteNameAttribute]
		require.True(suite.T(), ok)
		name, ok := attr.(string)
		require.True(suite.T(), ok)
		res[name] = node
	}
	return res
}

func (suite *ClusterGroupTestSuite) checkNodeParameters(node *ClusterNode, nodeIdx int, isClient bool, isLocal bool, order int64) {
	require.Equal(suite.T(), isClient, node.IsClient())
	require.Equal(suite.T(), isLocal, node.IsLocal())
	require.Equal(suite.T(), order, node.Order())
	require.Equal(suite.T(), []string{"127.0.0.1"}, node.Addresses())
	if !isClient {
		require.Equal(suite.T(), []string{"localhost"}, node.HostNames())
	}
	require.Equal(suite.T(), "srv_"+strconv.Itoa(nodeIdx), node.ConsistentId())
	suite.checkVersionInServerLogs(nodeIdx, node.Version())
}

func (suite *ClusterGroupTestSuite) checkVersionInServerLogs(nodeIdx int, version *Version) {
	logFiles, err := testing2.GetLogFiles(nodeIdx)
	require.NoError(suite.T(), err)
	reg, err := regexp.Compile(fmt.Sprintf("%d.%d.%d", version.Major(), version.Minor(), version.Maintenance()))
	require.NoError(suite.T(), err)
	isFound := false
	for _, logFile := range logFiles {
		isFound, err = testing2.MatchLog(reg, logFile)
		require.NoError(suite.T(), err)
		if isFound {
			break
		}
	}
	require.True(suite.T(), isFound)
}
