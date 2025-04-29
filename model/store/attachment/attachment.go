package store

import (
	"github.com/lyricat/goutils/model/core"
	"github.com/lyricat/goutils/model/store"
	"github.com/lyricat/goutils/model/store/attachment/dao"

	"gorm.io/gen"
)

func init() {
	store.RegistGenerate(
		gen.Config{
			OutPath: "model/store/attachment/dao",
		},
		func(g *gen.Generator) {
			g.ApplyInterface(func(core.AttachmentStore) {}, core.Attachment{})
		},
	)
}

func New(h *store.Handler) core.AttachmentStore {
	var q *dao.Query
	if !dao.Q.Available() {
		dao.SetDefault(h.DB)
		q = dao.Q
	} else {
		q = dao.Use(h.DB)
	}

	v, ok := interface{}(q.Attachment).(core.AttachmentStore)
	if !ok {
		panic("dao.Attachment is not core.AttachmentStore")
	}

	return &storeImpl{
		AttachmentStore: v,
	}
}

type storeImpl struct {
	core.AttachmentStore
}
