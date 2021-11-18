package southbound

import (
	"context"
	"sample-xapp/pkg/rnib"
	"sort"
	"strings"

	prototypes "github.com/gogo/protobuf/types"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/pdubuilder"
	e2smkpmv2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"
	"google.golang.org/protobuf/proto"
)

var log = logging.GetLogger("e2subscription")

const (
	kpmServiceModelOID = "1.3.6.1.4.1.53148.1.2.2.2"
	appID              = "sample-xapp"
)

type E2SubManager struct {
	E2Client   e2client.Client
	RnibClient rnib.TopoClient
	SMName     string
	SMVersion  string
}

func NewE2SubManager(smName string, smVersion string, e2tAddr string, e2tPort int) (*E2SubManager, error) {

	e2Client := e2client.NewClient(
		e2client.WithServiceModel(e2client.ServiceModelName(smName), e2client.ServiceModelVersion(smVersion)),
		e2client.WithAppID(appID),
		e2client.WithE2TAddress(e2tAddr, e2tPort),
	)

	rnibClient, err := rnib.NewTopoClient()
	if err != nil {
		return nil, err
	}

	return &E2SubManager{
		E2Client:   e2Client,
		RnibClient: *rnibClient,
		SMName:     smName,
		SMVersion:  smVersion,
	}, nil
}

func (e *E2SubManager) Start() error {
	go func() {
		err := e.watchE2Connections(context.Background())
		if err != nil {
			return
		}
	}()
	return nil
}

func (e *E2SubManager) watchE2Connections(ctx context.Context) error {
	ch := make(chan topoapi.Event)
	err := e.RnibClient.WatchE2Connections(ctx, ch)
	if err != nil {
		return err
	}

	for topoEvent := range ch {
		log.Infof("New topo event happens: %v", topoEvent)

		if topoEvent.Type == topoapi.EventType_ADDED || topoEvent.Type == topoapi.EventType_NONE {
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
			if !e.RnibClient.HasKPMRanFunction(ctx, e2NodeID, kpmServiceModelOID) {
				continue
			}

			go func() {
				err := e.newSubscription(ctx, e2NodeID)
				if err != nil {
					log.Warn(err)
				}
			}()

		} else if topoEvent.Type == topoapi.EventType_REMOVED {
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
			log.Infof("e2NodeID %v was disconnected", e2NodeID)
		}
	}

	return nil
}

func (e *E2SubManager) newSubscription(ctx context.Context, e2NodeID topoapi.ID) error {
	log.Info("Start creating a new subscription with e2NodeID %v", e2NodeID)
	aspects, err := e.RnibClient.GetE2NodeAspects(ctx, e2NodeID)
	if err != nil {
		return err
	}

	reportStyles, err := e.getReportStyle(aspects.ServiceModels)
	if err != nil {
		return err
	}

	cells, err := e.RnibClient.GetCells(ctx, e2NodeID)
	if err != nil {
		return err
	}

	eventTriggerData, err := e.CreateEventTriggerData(1000)
	if err != nil {
		return err
	}

	for _, reportStyle := range reportStyles {
		actions, err := e.createSubscriptionActions(ctx, reportStyle, cells, 1000)
		if err != nil {
			return err
		}

		ch := make(chan e2api.Indication)
		node := e.E2Client.Node(e2client.NodeID(e2NodeID))
		subName := "onos-kpimon-subscription"

		subSpec := e2api.SubscriptionSpec{
			Actions: actions,
			EventTrigger: e2api.EventTrigger{
				Payload: eventTriggerData,
			},
		}

		_, err = node.Subscribe(ctx, subName, subSpec, ch)
		if err != nil {
			return err
		}

		go func(indCh chan e2api.Indication) {
			for indication := range indCh {
				log.Infof("Received indication message: %v", indication)

				indHeader := e2smkpmv2.E2SmKpmIndicationHeader{}
				err := proto.Unmarshal(indication.Header, &indHeader)
				if err != nil {
					log.Warn(err)
				}
				indMessage := e2smkpmv2.E2SmKpmIndicationMessage{}
				err = proto.Unmarshal(indication.Payload, &indMessage)
				if err != nil {
					log.Warn(err)
				}

				log.Infof("Received indication message header: %v", indHeader.GetIndicationHeaderFormat1())
				log.Infof("Received indication message body: %v", indMessage.GetIndicationMessageFormat1())
			}
		}(ch)
	}

	return nil
}

func (e *E2SubManager) getReportStyle(serviceModelsInfo map[string]*topoapi.ServiceModelInfo) ([]*topoapi.KPMReportStyle, error) {
	for _, sm := range serviceModelsInfo {
		smName := strings.ToLower(sm.Name)
		if smName == string(e.SMName) && sm.OID == kpmServiceModelOID {
			kpmRanFunction := &topoapi.KPMRanFunction{}
			for _, ranFunction := range sm.RanFunctions {
				if ranFunction.TypeUrl == ranFunction.GetTypeUrl() {
					err := prototypes.UnmarshalAny(ranFunction, kpmRanFunction)
					if err != nil {
						return nil, err
					}
					return kpmRanFunction.ReportStyles, nil
				}
			}
		}
	}
	return nil, errors.New(errors.NotFound, "cannot retrieve report styles")
}

func (e *E2SubManager) CreateEventTriggerData(rtPeriod uint32) ([]byte, error) {
	e2SmKpmEventTriggerDefinition, err := pdubuilder.CreateE2SmKpmEventTriggerDefinition(rtPeriod)
	if err != nil {
		return []byte{}, err
	}

	err = e2SmKpmEventTriggerDefinition.Validate()
	if err != nil {
		return []byte{}, err
	}

	protoBytes, err := proto.Marshal(e2SmKpmEventTriggerDefinition)
	if err != nil {
		return []byte{}, err
	}

	return protoBytes, nil
}

func (e *E2SubManager) createSubscriptionActions(ctx context.Context, reportStyle *topoapi.KPMReportStyle, cells []*topoapi.E2Cell, granularity uint32) ([]e2api.Action, error) {
	actions := make([]e2api.Action, 0)

	sort.Slice(cells, func(i, j int) bool {
		return cells[i].CellObjectID < cells[j].CellObjectID
	})

	for index, cell := range cells {
		measInfoList := &e2smkpmv2.MeasurementInfoList{
			Value: make([]*e2smkpmv2.MeasurementInfoItem, 0),
		}

		for _, measurement := range reportStyle.Measurements {
			measTypeMeasName, err := pdubuilder.CreateMeasurementTypeMeasName(measurement.GetName())
			if err != nil {
				return nil, err
			}

			meanInfoItem, err := pdubuilder.CreateMeasurementInfoItem(measTypeMeasName, nil)
			if err != nil {
				return nil, err
			}
			measInfoList.Value = append(measInfoList.Value, meanInfoItem)
		}
		subID := int64(index + 1)
		actionDefinition, err := pdubuilder.CreateActionDefinitionFormat1(cell.GetCellObjectID(), measInfoList, granularity, subID)
		if err != nil {
			return nil, err
		}

		e2smKpmActionDefinition, err := pdubuilder.CreateE2SmKpmActionDefinitionFormat1(reportStyle.Type, actionDefinition)
		if err != nil {
			return nil, err
		}

		e2smKpmActionDefinitionProto, err := proto.Marshal(e2smKpmActionDefinition)
		if err != nil {
			return nil, err
		}

		action := &e2api.Action{
			ID:   int32(index),
			Type: e2api.ActionType_ACTION_TYPE_REPORT,
			SubsequentAction: &e2api.SubsequentAction{
				Type:       e2api.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
				TimeToWait: e2api.TimeToWait_TIME_TO_WAIT_ZERO,
			},
			Payload: e2smKpmActionDefinitionProto,
		}

		actions = append(actions, *action)

	}
	return actions, nil
}
