package ignite

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gitverse.ru/sbertech/ignite-go-client/internal"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=ClusterState

const (
	opClusterGetState        int16 = 5000
	opClusterChangeState     int16 = 5001
	opClusterChangeWalState  int16 = 5002
	opClusterGetWalState     int16 = 5003
	opClusterGroupGetNodeIds int16 = 5100
	opClusterGroupGetNodes   int16 = 5101
)

const (
	attributeFilterType int16 = iota + 1
	nodeTypeFilterType
)

// ClusterState represents state of the Ignite Cluster.
type ClusterState int8

const (
	// Inactive cluster state. In this state no cache operations are allowed. Node that, changing cluster state to Inactive leads to clearing all in-memory caches.
	Inactive ClusterState = iota
	// Active cluster state. In this state all cache operations are allowed.
	Active
	// ActiveReadOnly cluster state. In this state only cache read operations are allowed.
	ActiveReadOnly
)

// ClusterNode represents Ignite client or server node.
type ClusterNode struct {
	id           uuid.UUID
	consistentId interface{}
	attributes   map[string]interface{}
	addresses    []string
	hostNames    []string
	order        int64
	version      *Version
	isClient     bool
	isLocal      bool
}

// Id returns ID of the Ignite node.
func (n *ClusterNode) Id() uuid.UUID {
	return n.id
}

// ConsistentId returns consistent ID of the Ignite node.
func (n *ClusterNode) ConsistentId() interface{} {
	return n.consistentId
}

// Attributes returns attributes of the Ignite node.
func (n *ClusterNode) Attributes() map[string]interface{} {
	return n.attributes
}

// Addresses returns addresses of the Ignite node.
func (n *ClusterNode) Addresses() []string {
	return n.addresses
}

// HostNames returns host names of the Ignite node.
func (n *ClusterNode) HostNames() []string {
	return n.hostNames
}

// Order returns order of the Ignite node in topology.
func (n *ClusterNode) Order() int64 {
	return n.order
}

// Version returns version of the Ignite node.
func (n *ClusterNode) Version() *Version {
	return n.version
}

// IsClient returns true if the node is client, false otherwise.
func (n *ClusterNode) IsClient() bool {
	return n.isClient
}

// IsLocal returns true if current node is one to which Ignite Go Client is connected, false otherwise.
func (n *ClusterNode) IsLocal() bool {
	return n.isLocal
}

// String returns node's string representation.
func (n *ClusterNode) String() string {
	return fmt.Sprintf("ClusterNode [id=%v, consistentId=%v, addresses=%v, hostNames=%v, order=%v, version=%v, isClient=%t, isLocal=%t]",
		n.id, n.consistentId, n.addresses, n.hostNames, n.order, n.version, n.isClient, n.isLocal)
}

// Version represents Ignite version.
type Version struct {
	major             int8
	minor             int8
	maintenance       int8
	stage             string
	revisionTimestamp int64
	revisionHash      []byte
}

// Major returns major part of Ignite version.
func (v *Version) Major() int8 {
	return v.major
}

// Minor returns minor part of Ignite version.
func (v *Version) Minor() int8 {
	return v.minor
}

// Maintenance returns maintenance part of Ignite version.
func (v *Version) Maintenance() int8 {
	return v.maintenance
}

// DevelopmentStage returns current developments stage of the Ignite code running on server side.
func (v *Version) DevelopmentStage() string {
	return v.stage
}

// RevisionTimestamp returns build timestamp of Ignite code running on server side in seconds since January 1, 1970 UTC.
func (v *Version) RevisionTimestamp() int64 {
	return v.revisionTimestamp
}

// RevisionHash returns hashcode of Ignite code running on server side.
func (v *Version) RevisionHash() []byte {
	return v.revisionHash
}

// String returns string representation of the Version.
func (v *Version) String() string {
	timestamp := time.UnixMilli(v.RevisionTimestamp() * 1000).UTC().Format("20060102")
	hash := hex.EncodeToString(v.revisionHash)
	if len(hash) > 8 {
		hash = hash[:8]
	}
	return fmt.Sprintf("%d.%d.%d#%s-sha1:%s", v.major, v.minor, v.maintenance, timestamp, hash)
}

// ClusterGroup represents set of Ignite cluster nodes that meets the specified criteria. Criteria are defined by the set
// of node filters.
type ClusterGroup struct {
	channel    channel
	marshaller marshaller
	projection *projection
	topVer     atomic.Int64
	nodes      sync.Map
}

// Nodes returns cluster nodes that meets ClusterGroup criteria and are present in current Ignite topology.
func (g *ClusterGroup) Nodes(ctx context.Context) ([]*ClusterNode, error) {
	nodeIds, err := g.requestNodeIds(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := g.getOrRequestNodes(ctx, nodeIds)
	if err != nil {
		return nil, err
	}
	return g.projection.apply(ctx, nodes)
}

// Node returns ClusterNode with specified ID if it meets ClusterGroup criteria and is present in current Ignite topology.
func (g *ClusterGroup) Node(ctx context.Context, nodeId uuid.UUID) (*ClusterNode, error) {
	if ok := g.projection.isExcludedByFilters(nodeId); ok {
		return nil, nil
	}
	nodeIds, err := g.requestNodeIds(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := internal.ToSet(nodeIds)[nodeId]; !ok {
		return nil, nil
	}
	nodes, err := g.getOrRequestNodes(ctx, []uuid.UUID{nodeId})
	if err != nil {
		return nil, err
	}
	res, err := g.projection.apply(ctx, nodes)
	if err != nil {
		return nil, err
	}
	if len(res) != 0 {
		return res[0], nil
	}
	return nil, nil
}

// ForNodeIds returns ClusterGroup filter that filters out all nodes with an ID other than the ones specified.
func ForNodeIds(nodeIds ...uuid.UUID) func(_ *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		var filter clusterNodeFilter
		if len(nodeIds) == 0 {
			filter = nil
		} else {
			filter = &nodeIdFilter{nodeIds: internal.ToSet(nodeIds)}
		}
		return g.addProjectionFilter(filter)
	}
}

// ForAttribute returns a ClusterGroup filter that filters out all nodes that do not have an attribute with the
// specified name and value.
func ForAttribute(name string, val interface{}) func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(&nodeAttributesFilter{attributes: map[string]interface{}{name: val}})
	}
}

// ForClients returns ClusterGroup filter that filters out all nodes except clients.
func ForClients() func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(&nodeTypeFilter{isForClients: true})
	}
}

// ForServers returns ClusterGroup filter that filters out all nodes except servers.
func ForServers() func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(&nodeTypeFilter{isForClients: false})
	}
}

// ForPredicate returns ClusterGroup filter that filters out all nodes that don't match the specified predicate.
func ForPredicate(predicate func(node *ClusterNode) bool) func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(predicateBasedClientSideFilter(predicate))
	}
}

// ForRandom returns ClusterGroup filter that filters out all nodes except a single random node.
func ForRandom() func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(basicClientSideFilter(func(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
			if len(nodes) < 2 {
				return nodes, nil
			}
			rand.Seed(time.Now().UnixNano())
			pos := rand.Intn(len(nodes))
			return []*ClusterNode{nodes[pos]}, nil
		}))
	}
}

// ForYoungest returns ClusterGroup filter that filters out all nodes except the one with the maximum order in topology.
func ForYoungest() func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(basicClientSideFilter(func(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
			if len(nodes) < 2 {
				return nodes, nil
			}
			youngestNodeIdx := 0
			for idx, node := range nodes {
				if node.order > nodes[youngestNodeIdx].order {
					youngestNodeIdx = idx
				}
			}
			return []*ClusterNode{nodes[youngestNodeIdx]}, nil
		}))
	}
}

// ForOldest returns ClusterGroup filter that filters out all nodes except the one with the minimum order in topology.
func ForOldest() func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		return g.addProjectionFilter(basicClientSideFilter(func(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
			if len(nodes) < 2 {
				return nodes, nil
			}
			oldestNodeIdx := 0
			for idx, node := range nodes {
				if node.order < nodes[oldestNodeIdx].order {
					oldestNodeIdx = idx
				}
			}
			return []*ClusterNode{nodes[oldestNodeIdx]}, nil
		}))
	}
}

// ForClusterNodes returns ClusterGroup filter that filters out all nodes except the specified ones.
func ForClusterNodes(nodes ...*ClusterNode) func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		var nodeIds []uuid.UUID
		for _, node := range nodes {
			if node == nil {
				return errors.New("nil node")
			}
			nodeIds = append(nodeIds, node.Id())
		}

		err := ForNodeIds(nodeIds...)(g)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			g.nodes.Store(node.Id(), node)
		}
		return nil
	}
}

// ExceptNodes returns ClusterGroup filter that filters out all specified nodes.
func ExceptNodes(nodes ...*ClusterNode) func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		filteredNodeIds, err := internal.ToSetWithTransformer(nodes, func(node *ClusterNode) (uuid.UUID, error) {
			if node == nil {
				return uuid.UUID{}, errors.New("nil node")
			}
			return node.Id(), nil
		})
		if err != nil {
			return err
		}
		return g.addProjectionFilter(predicateBasedClientSideFilter(func(node *ClusterNode) bool {
			_, ok := filteredNodeIds[node.id]
			return !ok
		}))
	}
}

// ExceptGroupNodes returns ClusterGroup filter that filters out all nodes that meet criteria of the specified group.
func ExceptGroupNodes(group *ClusterGroup) func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		if group == nil {
			return errors.New("nil cluster group")
		}
		return g.addProjectionFilter(basicClientSideFilter(func(ctx context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
			grpNodes, err := group.Nodes(ctx)
			if err != nil {
				return nil, err
			}
			filteredNodeIds, _ := internal.ToSetWithTransformer(grpNodes, func(val *ClusterNode) (uuid.UUID, error) { return val.Id(), nil })
			return applyPredicate(nodes, func(node *ClusterNode) bool {
				_, ok := filteredNodeIds[node.id]
				return !ok
			})
		}))
	}
}

// ForHost returns ClusterGroup filter that filters out all nodes except ones that reside on the same host as the
// specified node.
func ForHost(node *ClusterNode) func(g *ClusterGroup) error {
	return func(g *ClusterGroup) error {
		if node == nil {
			return errors.New("nil cluster group")
		}
		return ForAttribute(attrMacsName, node.attributes[attrMacsName])(g)
	}
}

// ForHostNames returns ClusterGroup filter that filters out all nodes except ones that are running on the specified hosts.
func ForHostNames(hosts ...string) func(g *ClusterGroup) error {
	filteredHosts := internal.ToSet(hosts)
	return func(g *ClusterGroup) error {
		var filter clusterNodeFilter
		if len(filteredHosts) == 0 {
			filter = nil // nil means filter which produces no elements.
		} else {
			filter = predicateBasedClientSideFilter(func(node *ClusterNode) bool {
				for _, hostName := range node.hostNames {
					if _, ok := filteredHosts[hostName]; ok {
						return true
					}
				}
				return false
			})
		}
		return g.addProjectionFilter(filter)
	}
}

const attrMacsName = "org.apache.ignite.macs"

type clusterNodeFilter interface {
	apply(ctx context.Context, nodes []*ClusterNode) ([]*ClusterNode, error)
	merge(other clusterNodeFilter) (clusterNodeFilter, bool)
}

type clientSideNodeFiler interface {
	clusterNodeFilter
	isReorderingAllowed() bool
}

type serverSideNodeFilter interface {
	clusterNodeFilter
	write(ctx context.Context, out BinaryOutputStream, marshaller marshaller) error
}

var emptyProjection = &projection{
	[]clusterNodeFilter{
		basicClientSideFilter(func(_ context.Context, _ []*ClusterNode) ([]*ClusterNode, error) { return nil, nil }),
	},
}

func (g *ClusterGroup) getOrRequestNodes(ctx context.Context, nodeIds []uuid.UUID) ([]*ClusterNode, error) {
	res := make([]*ClusterNode, 0, len(nodeIds))
	var missingNodeIds []uuid.UUID

	for _, nodeId := range nodeIds {
		node, ok := g.nodes.Load(nodeId)
		if ok {
			res = append(res, node.(*ClusterNode))
		} else {
			missingNodeIds = append(missingNodeIds, nodeId)
		}
	}

	if len(missingNodeIds) != 0 {
		nodes, err := g.requestNodes(ctx, missingNodeIds)
		if err != nil {
			return nil, err
		}
		res = append(res, nodes...)
	}
	return res, nil
}

func (g *ClusterGroup) requestNodeIds(ctx context.Context) (nodeIds []uuid.UUID, err error) {
	var topVer int64
	g.channel.send(ctx, opClusterGroupGetNodeIds,
		func(currCh channel, out BinaryOutputStream) error {
			if !currCh.protocolContext().SupportsAttributeFeature(ClusterGroupsFeature) {
				return errors.New("cluster groups feature is not supported by the server")
			}
			out.WriteInt64(-1)
			if err := g.projection.writeServerSideFilters(ctx, out, g.marshaller); err != nil {
				return err
			}
			return nil
		},
		func(currCh channel, in BinaryInputStream, err0 error) {
			if err0 != nil {
				err = err0
				return
			}
			if err = ensureAvailable(in, boolBytes+longBytes); err != nil {
				return
			}
			if !in.ReadBool() {
				err = errors.New("unexpected server response for topology node IDs request")
				return
			}
			topVer = in.ReadInt64()
			nodeIds, err = readSlice(in, func(i int, in BinaryInputStream) (uuid.UUID, error) {
				return readUuid(in)
			})
		},
	)
	if err == nil && g.topVer.CompareAndSwap(g.topVer.Load(), topVer) {
		serverTopNodeIds := internal.ToSet(nodeIds)
		g.nodes.Range(func(val, _ interface{}) bool {
			id := val.(uuid.UUID)
			if _, ok := serverTopNodeIds[id]; !ok {
				g.nodes.Delete(id)
			}
			return true
		})
	}
	return
}

func (g *ClusterGroup) requestNodes(ctx context.Context, nodeIds []uuid.UUID) (nodes []*ClusterNode, err error) {
	g.channel.send(ctx, opClusterGroupGetNodes,
		func(currCh channel, out BinaryOutputStream) error {
			if !currCh.protocolContext().SupportsAttributeFeature(ClusterGroupsFeature) {
				return errors.New("cluster groups feature is not supported by the server")
			}
			return writeSequence(out, len(nodeIds), func(output BinaryOutputStream, idx int) error {
				writeUuid(out, &nodeIds[idx])
				return nil
			})
		},
		func(currCh channel, in BinaryInputStream, err0 error) {
			if err0 != nil {
				err = err0
				return
			}
			nodes, err = readSlice(in, func(i int, in BinaryInputStream) (*ClusterNode, error) {
				return readClusterNode(ctx, in, g.marshaller)
			})
		},
	)
	if err == nil {
		for _, node := range nodes {
			g.nodes.Store(node.Id(), node)
		}
	}
	return
}

func (g *ClusterGroup) addProjectionFilter(filter clusterNodeFilter) error {
	g.projection = appendFilter(g.projection, filter)
	return nil
}

func readClusterNode(ctx context.Context, in BinaryInputStream, marshaller marshaller) (*ClusterNode, error) {
	id, err := unmarshalUuid(in)
	if err != nil {
		return nil, err
	}
	attributes, err := readNodeAttributes(ctx, in, marshaller)
	if err != nil {
		return nil, err
	}
	addresses, err := readCollectionAsSlice[string](ctx, in, marshaller)
	if err != nil {
		return nil, err
	}
	hostNames, err := readCollectionAsSlice[string](ctx, in, marshaller)
	if err != nil {
		return nil, err
	}
	order := in.ReadInt64()
	isLocal := in.ReadBool()
	in.ReadBool() // Skip obsolete IsDaemon flag.
	isClient := in.ReadBool()
	consistentId, err := marshaller.unmarshal(ctx, in)
	if err != nil {
		return nil, err
	}
	version, err := readNodeVersion(in)
	if err != nil {
		return nil, err
	}

	sort.Strings(addresses)
	sort.Strings(hostNames)

	return &ClusterNode{
		id:           id,
		attributes:   attributes,
		addresses:    addresses,
		hostNames:    hostNames,
		order:        order,
		isLocal:      isLocal,
		isClient:     isClient,
		consistentId: consistentId,
		version:      version,
	}, nil
}

func readCollectionAsSlice[T any](ctx context.Context, in BinaryInputStream, marshaller marshaller) ([]T, error) {
	val, err := marshaller.unmarshal(ctx, in)
	if err != nil {
		return nil, err
	}
	col, ok := val.(Collection)
	if !ok {
		return nil, errors.New("unexpected value type")
	}
	slice, err := ToSlice[T](col)
	if err != nil {
		return nil, err
	}
	return slice, nil
}

func readNodeVersion(in BinaryInputStream) (*Version, error) {
	major := in.ReadInt8()
	minor := in.ReadInt8()
	maintenance := in.ReadInt8()
	stage, err := unmarshalString(in)
	if err != nil {
		return nil, err
	}
	revisionTimestamp := in.ReadInt64()
	revisionHash, err := unmarshalByteArray(in)
	if err != nil {
		return nil, err
	}
	return &Version{
		major:             major,
		minor:             minor,
		maintenance:       maintenance,
		stage:             stage,
		revisionTimestamp: revisionTimestamp,
		revisionHash:      revisionHash,
	}, nil
}

func readNodeAttributes(ctx context.Context, in BinaryInputStream, marshaller marshaller) (map[string]interface{}, error) {
	return readMap(in, func(stream BinaryInputStream) (string, interface{}, error) {
		key, err := unmarshalString(in)
		if err != nil {
			return "", nil, err
		}
		val, err := marshaller.unmarshal(ctx, in)
		if err != nil {
			return "", nil, err
		}
		return key, val, nil
	})
}

func applyPredicate(nodes []*ClusterNode, test func(node *ClusterNode) bool) ([]*ClusterNode, error) {
	res := make([]*ClusterNode, 0)
	for _, node := range nodes {
		if test(node) {
			res = append(res, node)
		}
	}
	return res, nil
}

func isReorderingAllowed(filter clusterNodeFilter) bool {
	if clientSideFilter, ok := filter.(clientSideNodeFiler); ok {
		return clientSideFilter.isReorderingAllowed()
	}
	return true
}

func checkClusterApiSupportedByServer(protoCtx *ProtocolContext) error {
	if !protoCtx.SupportsClusterApi() && protoCtx.SupportsAttributeFeature(ClusterStatesFeature) {
		return errors.New("cluster API is not supported by the server")
	}
	return nil
}

// projection represents set of filters that allows to define which nodes are belong to the particular cluster group.
type projection struct {
	filters []clusterNodeFilter
}

func appendFilter(proj *projection, filter clusterNodeFilter) *projection {
	// filter == nil means that this filter produces no elements.
	if proj == emptyProjection || filter == nil {
		return emptyProjection
	}

	if isReorderingAllowed(filter) {
		for i := len(proj.filters) - 1; i >= 0; i-- {
			prev := proj.filters[i]
			if !isReorderingAllowed(prev) {
				break
			}

			if filter, ok := prev.merge(filter); ok {
				if filter == nil {
					return emptyProjection
				}
				proj.filters[i] = filter
				return proj
			}
		}
	}
	proj.filters = append(proj.filters, filter)
	return proj
}

func (p *projection) apply(ctx context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}
	var err error
	for _, filter := range p.filters {
		nodes, err = filter.apply(ctx, nodes)
		if err != nil {
			return nil, err
		}
		if len(nodes) == 0 {
			break
		}
	}
	return nodes, nil
}

func (p *projection) isExcludedByFilters(nodeId uuid.UUID) bool {
	idFilter := p.findSingleNodeIdFilter()
	if idFilter != nil {
		_, ok := idFilter.nodeIds[nodeId]
		return !ok
	}
	return false
}

func (p *projection) findSingleNodeIdFilter() *nodeIdFilter {
	if len(p.filters) == 1 {
		if idFilter, ok := p.filters[0].(*nodeIdFilter); ok {
			return idFilter
		}
	}
	return nil
}

func (p *projection) writeServerSideFilters(ctx context.Context, out BinaryOutputStream, marshaller marshaller) error {
	out.EnsureAvailable(intBytes)
	startPos := out.Position()
	out.SetPosition(startPos + intBytes)

	var filtersCnt int32 = 0
	for _, filter := range p.filters {
		if !isReorderingAllowed(filter) {
			break
		}
		if serverFilter, ok := filter.(serverSideNodeFilter); ok {
			err := serverFilter.write(ctx, out, marshaller)
			if err != nil {
				return err
			}
			filtersCnt++
		}
	}

	retPos := out.Position()
	out.SetPosition(startPos)
	out.WriteInt32(filtersCnt)
	out.SetPosition(retPos)
	return nil
}

// basicClientSideFilter clientSideNodeFiler that cannot be reordered.
type basicClientSideFilter func(ctx context.Context, nodes []*ClusterNode) ([]*ClusterNode, error)

func (f basicClientSideFilter) apply(ctx context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
	return f(ctx, nodes)
}

func (f basicClientSideFilter) merge(clusterNodeFilter) (clusterNodeFilter, bool) {
	return nil, false
}

func (f basicClientSideFilter) isReorderingAllowed() bool {
	return false
}

// predicateBasedClientSideFilter represents clientSideNodeFiler implementation that filters out nodes based on
// specified predicate.
type predicateBasedClientSideFilter func(node *ClusterNode) bool

func (f predicateBasedClientSideFilter) apply(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
	return applyPredicate(nodes, f)
}

func (f predicateBasedClientSideFilter) merge(clusterNodeFilter) (clusterNodeFilter, bool) {
	return nil, false
}

func (f predicateBasedClientSideFilter) isReorderingAllowed() bool {
	return true
}

// nodeTypeFilter represents serverSideNodeFilter implementation that filters out nodes by their type - client/server.
type nodeTypeFilter struct {
	isForClients bool
}

func (f *nodeTypeFilter) apply(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
	return applyPredicate(nodes, func(node *ClusterNode) bool { return node.isClient == f.isForClients })
}

func (f *nodeTypeFilter) merge(other clusterNodeFilter) (clusterNodeFilter, bool) {
	if otherFilter, ok := other.(*nodeTypeFilter); ok {
		if otherFilter.isForClients == f.isForClients {
			return f, true
		} else {
			return nil, true // nil means filter which produces no elements.
		}
	}
	return nil, false
}

func (f *nodeTypeFilter) write(_ context.Context, out BinaryOutputStream, _ marshaller) error {
	out.WriteInt16(nodeTypeFilterType)
	out.WriteBool(!f.isForClients)
	return nil
}

// nodeAttributesFilter represents serverSideNodeFilter implementation that filters out nodes by specified attributes.
type nodeAttributesFilter struct {
	attributes map[string]interface{}
}

func (f *nodeAttributesFilter) apply(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
	return applyPredicate(nodes, func(node *ClusterNode) bool {
		for name, val := range f.attributes {
			if nodeAttributeVal, ok := node.attributes[name]; !ok || val != nodeAttributeVal {
				return false
			}
		}

		return true
	})
}

func (f *nodeAttributesFilter) merge(other clusterNodeFilter) (clusterNodeFilter, bool) {
	if otherFilter, ok := other.(*nodeAttributesFilter); ok {
		for otherKey, otherVal := range otherFilter.attributes {
			if val, ok := f.attributes[otherKey]; ok {
				if val != otherVal {
					return nil, true // nil means filter which produces no elements.
				}
			}
			f.attributes[otherKey] = otherVal
		}
		return f, true
	}
	return nil, false
}

func (f *nodeAttributesFilter) write(ctx context.Context, out BinaryOutputStream, marshaller marshaller) error {
	for name, val := range f.attributes {
		out.WriteInt16(attributeFilterType)
		marshalString(out, name)
		err := marshaller.marshal(ctx, out, val)
		if err != nil {
			return err
		}
	}
	return nil
}

// nodeIdFilter represents clientSideNodeFiler that filters out nodes by their IDs.
type nodeIdFilter struct {
	nodeIds map[uuid.UUID]interface{}
}

func (f *nodeIdFilter) apply(_ context.Context, nodes []*ClusterNode) ([]*ClusterNode, error) {
	return applyPredicate(nodes, func(node *ClusterNode) bool {
		_, ok := f.nodeIds[node.id]
		return ok
	})
}

func (f *nodeIdFilter) merge(other clusterNodeFilter) (clusterNodeFilter, bool) {
	if otherFilter, ok := other.(*nodeIdFilter); ok {
		res := make(map[uuid.UUID]interface{})
		for otherId := range otherFilter.nodeIds {
			if _, ok := f.nodeIds[otherId]; ok {
				res[otherId] = nil
			}
		}

		if len(res) == 0 {
			return nil, true // nil means filter which produces no elements.
		} else {
			f.nodeIds = res
			return f, true
		}
	}
	return nil, false
}

func (f *nodeIdFilter) isReorderingAllowed() bool {
	return true
}
