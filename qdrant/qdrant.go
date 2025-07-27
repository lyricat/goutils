package qdrant

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type (
	QdrantPoint struct {
		ID      int64                `json:"id"`
		UUID    string               `json:"uuid"`
		Score   float32              `json:"score"`
		Vector  []float32            `json:"vector"`
		Payload map[string]*pb.Value `json:"payload"`
	}

	Config struct {
		Addr   string
		APIKey string
	}

	QdrantClient struct {
		Addr   string
		APIKey string
		ColCli pb.CollectionsClient
		Conn   *grpc.ClientConn
	}

	CommonParams struct {
		CollectionName string
		PointID        uint64
		PointUUID      string
		Selector       *pb.PointsSelector
	}

	GetPointsParams struct {
		CommonParams
	}

	UpsertPointsParams struct {
		CommonParams
		Vector     []float32
		Payload    map[string]UpsertPointPayloadItem
		WaitUpsert bool
	}

	UpsertPointPayloadItem struct {
		Type  string
		Value any
	}

	DeletePointsParams struct {
		CommonParams
	}

	SearchPointsParams struct {
		CollectionName string
		Vector         []float32
		TopK           uint64
		Key            string
		Value          int64
		Offset         uint64
		Filter         *pb.Filter
	}

	CreateCollectionParams struct {
		CollectionName string
		VectorSize     uint64
		Indexes        []CreateCollectionIndexItem
	}

	CreateCollectionIndexItem struct {
		Name string
		Type string
	}
)

func (p *CommonParams) GetPbPointIDs() ([]*pb.PointId, error) {
	ids := make([]*pb.PointId, 0)
	if p.PointID > 0 {
		ids = append(ids, &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: p.PointID}})
	} else if p.PointUUID != "" {
		ids = append(ids, &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: p.PointUUID}})
	} else {
		slog.Error("point ID or UUID is required")
		return nil, fmt.Errorf("point ID or UUID is required")
	}
	return ids, nil
}

func (p *CommonParams) GetPbSelector() (*pb.PointsSelector, error) {
	ids, err := p.GetPbPointIDs()
	if err != nil {
		return nil, err
	}

	var selector *pb.PointsSelector
	if len(ids) != 0 {
		selector = &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: ids,
				},
			},
		}
	} else if p.Selector != nil {
		selector = p.Selector
	}
	return selector, nil
}

func (p *CommonParams) GetPbPointID() (*pb.PointId, error) {
	var id *pb.PointId
	if p.PointID > 0 {
		id = &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: p.PointID}}
	} else if p.PointUUID != "" {
		id = &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: p.PointUUID}}
	} else {
		slog.Error("point ID or UUID is required")
		return nil, fmt.Errorf("point ID or UUID is required")
	}
	return id, nil
}

func genInterceptor(apiKey string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, "api-key", apiKey)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

func New(cfg Config) *QdrantClient {
	config := &tls.Config{}
	interceptor := genInterceptor(cfg.APIKey)
	conn, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(credentials.NewTLS(config)), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		slog.Error("did not connect", "error", err)
		panic(err)
	}

	ColCli := pb.NewCollectionsClient(conn)

	return &QdrantClient{
		Addr:   cfg.Addr,
		APIKey: cfg.APIKey,
		ColCli: ColCli,
		Conn:   conn,
	}
}

func (c *QdrantClient) Check() (string, error) {
	qdrantClient := pb.NewQdrantClient(c.Conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	healthCheckResult, err := qdrantClient.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		slog.Error("could not get health", "error", err)
		return "", err
	}
	return healthCheckResult.GetVersion(), nil
}

func (c *QdrantClient) Close() {
	c.Conn.Close()
}

func (c *QdrantClient) GetPoints(ctx context.Context, params GetPointsParams) (*QdrantPoint, error) {
	ids, err := params.GetPbPointIDs()
	if err != nil {
		return nil, err
	}

	pointsClient := pb.NewPointsClient(c.Conn)
	pointsById, err := pointsClient.Get(ctx, &pb.GetPoints{
		CollectionName: params.CollectionName,
		Ids:            ids,
		WithVectors:    &pb.WithVectorsSelector{SelectorOptions: &pb.WithVectorsSelector_Enable{Enable: true}},
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		slog.Error("could not retrieve points", "error", err)
		return nil, err
	}
	result := pointsById.GetResult()
	if len(result) > 0 {
		qp := &QdrantPoint{}
		qp.LoadFromRetrievedPoint(result[0])
		return qp, nil
	}
	return nil, nil
}

func (c *QdrantClient) UpsertPoints(ctx context.Context, params UpsertPointsParams) error {
	payload := make(map[string]*pb.Value)
	for k, v := range params.Payload {
		switch v.Type {
		case "int":
			vint := int64(v.Value.(int64))
			payload[k] = &pb.Value{
				Kind: &pb.Value_IntegerValue{IntegerValue: vint},
			}
		case "uint":
			vint := int64(v.Value.(uint64))
			payload[k] = &pb.Value{
				Kind: &pb.Value_IntegerValue{IntegerValue: vint},
			}
		case "double":
			payload[k] = &pb.Value{
				Kind: &pb.Value_DoubleValue{DoubleValue: v.Value.(float64)},
			}
		case "bool":
			payload[k] = &pb.Value{
				Kind: &pb.Value_BoolValue{BoolValue: v.Value.(bool)},
			}
		case "text":
			payload[k] = &pb.Value{
				Kind: &pb.Value_StringValue{StringValue: v.Value.(string)},
			}
		}
	}

	waitUpsert := params.WaitUpsert

	id, err := params.GetPbPointID()
	if err != nil {
		return err
	}

	upsertPoints := []*pb.PointStruct{
		{
			Id:      id,
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: params.Vector}}},
			Payload: payload,
		},
	}

	pointsClient := pb.NewPointsClient(c.Conn)

	if _, err := pointsClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: params.CollectionName,
		Wait:           &waitUpsert,
		Points:         upsertPoints,
	}); err != nil {
		slog.Error("could not upsert points", "error", err)
	}

	return nil
}

func (c *QdrantClient) DeletePoints(ctx context.Context, params DeletePointsParams) error {
	selector, err := params.GetPbSelector()
	if err != nil {
		return err
	}

	pointsClient := pb.NewPointsClient(c.Conn)
	if _, err := pointsClient.Delete(ctx, &pb.DeletePoints{
		CollectionName: params.CollectionName,
		Points:         selector,
	}); err != nil {
		slog.Error("could not delete points", "error", err)
	}

	return nil
}

func (c *QdrantClient) SearchPointsWithFilter(ctx context.Context, params SearchPointsParams) ([]*QdrantPoint, error) {
	filter := &pb.Filter{}
	if params.Filter != nil {
		filter = params.Filter
	} else {
		if params.Key != "" {
			filter = &pb.Filter{
				Must: []*pb.Condition{
					{
						ConditionOneOf: &pb.Condition_Field{
							Field: &pb.FieldCondition{
								Key: params.Key,
								Match: &pb.Match{
									MatchValue: &pb.Match_Integer{
										Integer: params.Value,
									},
								},
							},
						},
					},
				},
			}
		}
	}
	pointsClient := pb.NewPointsClient(c.Conn)
	filteredSearchResult, err := pointsClient.Search(ctx, &pb.SearchPoints{
		CollectionName: params.CollectionName,
		Vector:         params.Vector,
		Limit:          params.TopK,
		Offset:         &params.Offset,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		slog.Error("could not search points", "error", err)
	}

	result := filteredSearchResult.GetResult()
	qpList := make([]*QdrantPoint, 0)
	for _, p := range result {
		qp := &QdrantPoint{}
		qp.LoadFromScoredPoint(p)
		qpList = append(qpList, qp)
	}
	return qpList, nil
}

func (c *QdrantClient) CreateCollection(ctx context.Context, params CreateCollectionParams) error {
	// Create new collection
	var defaultSegmentNumber uint64 = 2
	cols := []string{params.CollectionName}
	for _, collectionName := range cols {
		_, err := c.ColCli.Create(ctx, &pb.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: &pb.VectorsConfig{Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     params.VectorSize,
					Distance: pb.Distance_Dot,
				},
			}},
			OptimizersConfig: &pb.OptimizersConfigDiff{
				DefaultSegmentNumber: &defaultSegmentNumber,
			},
		})
		if err != nil {
			slog.Error("could not create collection", "error", err)
			return err
		}
		// Create string field index
		pointsClient := pb.NewPointsClient(c.Conn)
		for _, indexItem := range params.Indexes {
			fieldType := pb.FieldType_FieldTypeKeyword
			switch indexItem.Type {
			case "int":
				fieldType = pb.FieldType_FieldTypeInteger
			case "float":
				fieldType = pb.FieldType_FieldTypeFloat
			case "bool":
				fieldType = pb.FieldType_FieldTypeBool
			case "text":
				fieldType = pb.FieldType_FieldTypeText
			case "keyword":
				fieldType = pb.FieldType_FieldTypeKeyword
			case "date":
				fieldType = pb.FieldType_FieldTypeDatetime
			case "geo":
				fieldType = pb.FieldType_FieldTypeGeo
			case "uuid":
				fieldType = pb.FieldType_FieldTypeUuid
			}

			_, err := pointsClient.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
				CollectionName: collectionName,
				FieldName:      indexItem.Name,
				FieldType:      &fieldType,
			})

			if err != nil {
				slog.Error("could not create index", "field", indexItem.Name, "error", err)
				return err
			}
		}
	}

	return nil
}

func (qp *QdrantPoint) LoadFromRetrievedPoint(p *pb.RetrievedPoint) error {
	qp.ID = int64(p.Id.GetNum())
	qp.UUID = p.Id.GetUuid()
	qp.Score = 0
	qp.Vector = p.GetVectors().GetVector().GetData()
	payload := p.GetPayload()
	qp.Payload = make(map[string]*pb.Value)
	for k, v := range payload {
		qp.Payload[k] = v
	}
	return nil
}

func (qp *QdrantPoint) LoadFromScoredPoint(p *pb.ScoredPoint) error {
	qp.ID = int64(p.Id.GetNum())
	qp.UUID = p.Id.GetUuid()
	qp.Score = p.Score
	qp.Vector = p.GetVectors().GetVector().GetData()
	payload := p.GetPayload()
	qp.Payload = make(map[string]*pb.Value)
	for k, v := range payload {
		qp.Payload[k] = v
	}
	return nil
}
