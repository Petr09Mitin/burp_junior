package mongo_repo

import (
	"context"

	"github.com/burp_junior/customerrors"
	"github.com/burp_junior/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Responses struct {
	Col *mongo.Collection
}

func NewResponsesRepo(col *mongo.Collection) (r *Responses) {
	return &Responses{
		Col: col,
	}
}

func (r *Responses) SaveResponse(ctx context.Context, resp *domain.HTTPResponse) (savedResp *domain.HTTPResponse, err error) {
	result, err := r.Col.InsertOne(context.Background(), resp)
	if err != nil {
		err = customerrors.ErrInternal
		return
	}

	resp.ID = result.InsertedID.(primitive.ObjectID).Hex()
	savedResp = resp

	return
}
