package mongo_repo

import (
	"context"

	"github.com/burp_junior/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Requests struct {
	Col *mongo.Collection
}

func NewRequestsRepo(col *mongo.Collection) (r *Requests) {
	return &Requests{
		Col: col,
	}
}

func (r *Requests) SaveRequest(ctx context.Context, req *domain.HTTPRequest) (savedReq *domain.HTTPRequest, err error) {
	result, err := r.Col.InsertOne(context.Background(), req)
	if err != nil {
		return
	}

	req.ID = result.InsertedID.(primitive.ObjectID).Hex()
	savedReq = req

	return
}

func (r *Requests) GetRequestsList(ctx context.Context) (reqs []*domain.HTTPRequest, err error) {
	reqs = make([]*domain.HTTPRequest, 0)

	cursor, err := r.Col.Find(context.Background(), primitive.M{})
	if err != nil {
		return
	}

	for cursor.Next(context.Background()) {
		var req domain.HTTPRequest
		err = cursor.Decode(&req)
		if err != nil {
			return
		}

		reqs = append(reqs, &req)
	}

	return
}

func (r *Requests) GetRequestByID(ctx context.Context, id string) (req *domain.HTTPRequest, err error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return
	}

	err = r.Col.FindOne(context.Background(), primitive.M{"_id": objID}).Decode(&req)
	if err != nil {
		return
	}

	return
}
