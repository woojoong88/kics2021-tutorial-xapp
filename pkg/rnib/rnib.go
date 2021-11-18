package rnib

import (
	"context"
	"fmt"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
)

var log = logging.GetLogger("rnib")

type TopoClient struct {
	Client toposdk.Client
}

func NewTopoClient() (*TopoClient, error) {
	sdkClient, err := toposdk.NewClient()
	if err != nil {
		return nil, err
	}

	return &TopoClient{
		Client: sdkClient,
	}, nil
}

func (t *TopoClient) WatchE2Connections(ctx context.Context, ch chan topoapi.Event) error {
	err := t.Client.Watch(ctx, ch, toposdk.WithWatchFilters(getControlRelationFilter()))

	if err != nil {
		return err
	}
	return nil
}

func (t *TopoClient) HasKPMRanFunction(ctx context.Context, nodeID topoapi.ID, oid string) bool {
	e2Node, err := t.GetE2NodeAspects(ctx, nodeID)
	if err != nil {
		log.Warn(err)
		return false
	}

	for _, sm := range e2Node.GetServiceModels() {
		if sm.OID == oid {
			return true
		}
	}

	return false
}

func (t *TopoClient) GetE2NodeAspects(ctx context.Context, nodeID topoapi.ID) (*topoapi.E2Node, error) {
	object, err := t.Client.Get(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	e2Node := &topoapi.E2Node{}
	object.GetAspect(e2Node)
	return e2Node, err

}

func (t *TopoClient) GetCells(ctx context.Context, nodeID topoapi.ID) ([]*topoapi.E2Cell, error) {
	filter := &topoapi.Filters{
		RelationFilter: &topoapi.RelationFilter{SrcId: string(nodeID),
			RelationKind: topoapi.CONTAINS,
			TargetKind:   ""}}

	objects, err := t.Client.List(ctx, toposdk.WithListFilters(filter))
	if err != nil {
		return nil, err
	}
	var cells []*topoapi.E2Cell
	for _, obj := range objects {
		targetEntity := obj.GetEntity()
		if targetEntity.GetKindID() == topoapi.E2CELL {
			cellObject := &topoapi.E2Cell{}
			obj.GetAspect(cellObject)
			cells = append(cells, cellObject)
		}
	}

	if len(cells) == 0 {
		return nil, fmt.Errorf("there is no cell")
	}

	return cells, nil
}

func getControlRelationFilter() *topoapi.Filters {
	controlRelationFilter := &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: topoapi.CONTROLS,
				},
			},
		},
	}
	return controlRelationFilter
}
