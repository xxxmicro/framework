package gorm

import (
	"context"
	"errors"
	"fmt"

	_gorm "github.com/jinzhu/gorm"
	"github.com/xxxmicro/framework/database/gorm/opentracing"
	"github.com/xxxmicro/framework/domain/model"
	"github.com/xxxmicro/framework/domain/repository"
	breflect "github.com/xxxmicro/framework/reflect"
)

type BaseRepository struct {
	DB *_gorm.DB
}

func NewBaseRepository(db *_gorm.DB) repository.BaseRepository {
	return &BaseRepository{db}
}

func (r *BaseRepository) Create(c context.Context, m model.Model) error {
	db := opentracing.SetSpanToGorm(c, r.DB)

	return db.Create(m).Error
}

func (r *BaseRepository) Upsert(c context.Context, m model.Model) (*repository.ChangeInfo, error) {
	db := opentracing.SetSpanToGorm(c, r.DB)

	result := db.Save(m)
	if result.Error != nil {
		return nil, result.Error
	}

	change := &repository.ChangeInfo{
		Updated: int(result.RowsAffected),
	}
	return change, nil
}

func (r *BaseRepository) Update(c context.Context, m model.Model, data interface{}) error {
	db := opentracing.SetSpanToGorm(c, r.DB)

	// 主键保护，如果 m 什么都没设置，这里将会删除表的所有记录
	scope := r.DB.NewScope(m)
	if scope.PrimaryKeyZero() {
		return errors.New(fmt.Sprintf("primary key(%s) must be set for update", scope.PrimaryKey()))
	}

	return db.Model(m).Update(data).Error
}

func (r *BaseRepository) FindOne(c context.Context, m model.Model) error {
	db := opentracing.SetSpanToGorm(c, r.DB)

	return db.Where(m.Unique()).Take(m).Error
}

func (r *BaseRepository) Delete(c context.Context, m model.Model) error {
	// 主键保护，如果 m 什么都没设置，这里将会删除表的所有记录
	ms := r.DB.NewScope(m).GetModelStruct()
	for _, pf := range ms.PrimaryFields {
		value, err := breflect.GetStructField(m, pf.Name)
		if err != nil {
			return err
		}

		if breflect.IsBlank(value) {
			return errors.New(fmt.Sprintf("primary key %s must set for delete", pf.Name))
		}
	}

	return r.DB.Delete(m).Error
}

func (r *BaseRepository) Page(c context.Context, m model.Model, query *model.PageQuery, resultPtr interface{}) (total int, pageCount int, err error) {
	// items := breflect.MakeSlicePtr(m, 0, 0)
	ms := r.DB.NewScope(m).GetModelStruct()

	dbHandler := r.DB.Model(m)
	dbHandler, err = buildQuery(dbHandler, ms, query.Filters)
	if err != nil {
		return
	}

	dbHandler, err = buildSort(dbHandler, ms, query.Sort)
	if err != nil {
		return
	}

	total, pageCount, err = pageQuery(dbHandler, query.PageNo, query.PageSize, resultPtr)

	return
}

func (r *BaseRepository) Cursor(c context.Context, query *model.CursorQuery, m model.Model, resultPtr interface{}) (extra *model.CursorExtra, err error) {
	ms := r.DB.NewScope(m).GetModelStruct()

	dbHandler := r.DB.Model(m)
	dbHandler, err = buildQuery(dbHandler, ms, query.Filters)
	if err != nil {
		return
	}

	dbHandler, reverse, err := gormCursorFilter(dbHandler, ms, query)
	if err != nil {
		return
	}

	// items := breflect.MakeSlicePtr(m, 0, 0)

	if err = dbHandler.Limit(query.Size).Find(resultPtr).Error; err != nil {
		return
	}

	if reverse {
		breflect.SlicePtrReverse(resultPtr)
	}

	var minCursor interface{} = nil
	var maxCursor interface{} = nil

	count := breflect.SlicePtrLen(resultPtr)
	if count > 0 {
		minItem := breflect.SlicePtrIndexOf(resultPtr, 0)
		field, ok := FindField(query.CursorSort.Property, ms, dbHandler)
		if !ok {
			err = errors.New("field not found")
			return
		}

		minCursor, err = breflect.GetStructField(minItem, field.Name)
		if err != nil {
			return
		}

		maxItem := breflect.SlicePtrIndexOf(resultPtr, count-1)
		maxCursor, err = breflect.GetStructField(maxItem, field.Name)
		if err != nil {
			return
		}
	}

	extra = &model.CursorExtra{
		Direction: query.Direction,
		Size:      query.Size,
		HasMore:   count == query.Size,
		MinCursor: minCursor,
		MaxCursor: maxCursor,
	}

	return
}
