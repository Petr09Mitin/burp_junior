package mongo

import "go.mongodb.org/mongo-driver/mongo"

type Requests struct {
	Col *mongo.Collection
}

func NewRequestsRepo(col *mongo.Collection) (r *Requests) {
	return &Requests{
		Col: col,
	}
}
