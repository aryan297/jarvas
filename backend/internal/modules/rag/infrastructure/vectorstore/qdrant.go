package vectorstore

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	ragport "github.com/jarvas/backend/internal/modules/rag/application/port"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type QdrantStore struct {
	collections pb.CollectionsClient
	points      pb.PointsClient
	apiKey      string
}

func NewQdrantStore(host string, port int, apiKey string) (*QdrantStore, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant dial: %w", err)
	}
	logger.Info("qdrant connected", zap.String("addr", addr))
	return &QdrantStore{
		collections: pb.NewCollectionsClient(conn),
		points:      pb.NewPointsClient(conn),
		apiKey:      apiKey,
	}, nil
}

// EnsureCollection creates the collection if it does not exist.
func (q *QdrantStore) EnsureCollection(ctx context.Context, name string, dim uint64) error {
	ctx = q.withAuth(ctx)
	_, err := q.collections.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
	if err == nil {
		return nil // already exists
	}

	distance := pb.Distance_Cosine
	_, err = q.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     dim,
					Distance: distance,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant create collection %q: %w", name, err)
	}
	logger.Info("qdrant collection created", zap.String("name", name), zap.Uint64("dim", dim))
	return nil
}

// Upsert stores a batch of vectors with their payloads.
func (q *QdrantStore) Upsert(ctx context.Context, collection string, points []ragport.VectorPoint) error {
	ctx = q.withAuth(ctx)
	pbPoints := make([]*pb.PointStruct, 0, len(points))
	for _, p := range points {
		pbPoints = append(pbPoints, &pb.PointStruct{
			Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: p.ID.String()}},
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: p.Vector}}},
			Payload: toPayload(p.Payload),
		})
	}

	wait := true
	_, err := q.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collection,
		Points:         pbPoints,
		Wait:           &wait,
	})
	return err
}

// Search finds the top-k nearest vectors filtered by payload fields.
func (q *QdrantStore) Search(ctx context.Context, collection string, query []float32, filter map[string]string, topK uint64, minScore float32) ([]ragport.SearchResult, error) {
	ctx = q.withAuth(ctx)

	var pbFilter *pb.Filter
	if len(filter) > 0 {
		var conditions []*pb.Condition
		for k, v := range filter {
			conditions = append(conditions, &pb.Condition{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: k,
						Match: &pb.Match{
							MatchValue: &pb.Match_Keyword{Keyword: v},
						},
					},
				},
			})
		}
		pbFilter = &pb.Filter{Must: conditions}
	}

	withPayload := &pb.WithPayloadSelector{
		SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true},
	}

	resp, err := q.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collection,
		Vector:         query,
		Limit:          topK,
		Filter:         pbFilter,
		WithPayload:    withPayload,
		ScoreThreshold: &minScore,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search: %w", err)
	}

	results := make([]ragport.SearchResult, 0, len(resp.Result))
	for _, r := range resp.Result {
		id, _ := uuid.Parse(r.Id.GetUuid())
		payload := fromPayload(r.Payload)
		results = append(results, ragport.SearchResult{
			ID:      id,
			Score:   r.Score,
			Payload: payload,
		})
	}
	return results, nil
}

// DeleteByFilter removes all points matching the given payload filters.
func (q *QdrantStore) DeleteByFilter(ctx context.Context, collection string, filter map[string]string) error {
	ctx = q.withAuth(ctx)

	var conditions []*pb.Condition
	for k, v := range filter {
		conditions = append(conditions, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key:   k,
					Match: &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: v}},
				},
			},
		})
	}

	wait := true
	_, err := q.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: collection,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{Must: conditions},
			},
		},
		Wait: &wait,
	})
	return err
}

func (q *QdrantStore) withAuth(ctx context.Context) context.Context {
	if q.apiKey == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "api-key", q.apiKey)
}

// ── Payload helpers ───────────────────────────────────────────────────────────

func toPayload(m map[string]interface{}) map[string]*pb.Value {
	out := make(map[string]*pb.Value, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			out[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: val}}
		case int:
			out[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: int64(val)}}
		case int64:
			out[k] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: val}}
		case float64:
			out[k] = &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: val}}
		case bool:
			out[k] = &pb.Value{Kind: &pb.Value_BoolValue{BoolValue: val}}
		}
	}
	return out
}

func fromPayload(m map[string]*pb.Value) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch val := v.Kind.(type) {
		case *pb.Value_StringValue:
			out[k] = val.StringValue
		case *pb.Value_IntegerValue:
			out[k] = val.IntegerValue
		case *pb.Value_DoubleValue:
			out[k] = val.DoubleValue
		case *pb.Value_BoolValue:
			out[k] = val.BoolValue
		}
	}
	return out
}
