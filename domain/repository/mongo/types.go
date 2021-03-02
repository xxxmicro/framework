package mongo

import (
	"github.com/xxxmicro/framework/domain/model"
	"gopkg.in/mgo.v2"
)

type Indexed interface {
	Indexes() []mgo.Index
	model.Model
}
