/**
 * Copyright 2014 @ S1N1 Team.
 * name :
 * author : jarryliu
 * date : 2013-12-08 11:09
 * description :
 * history :
 */

package repository

import (
	"database/sql"
	"fmt"
	"github.com/atnet/gof/db"
	"go2o/src/core/domain/interface/sale"
	saleImpl "go2o/src/core/domain/sale"
	"go2o/src/core/infrastructure/log"
	"github.com/atnet/gof/algorithm/iterator"
	"go2o/src/core/infrastructure/format"
)

var _ sale.ISaleRep = new(saleRep)

type saleRep struct {
	db.Connector
	cache   map[int]sale.ISale
	_tagRep sale.ISaleTagRep
}

func NewSaleRep(c db.Connector, saleTagRep sale.ISaleTagRep) sale.ISaleRep {
	return (&saleRep{
		Connector: c,
		_tagRep:   saleTagRep,
	}).init()
}

func (this *saleRep) init() sale.ISaleRep {
	this.cache = make(map[int]sale.ISale)
	return this
}

func (this *saleRep) GetSale(partnerId int) sale.ISale {
	v, ok := this.cache[partnerId]
	if !ok {
		v = saleImpl.NewSale(partnerId, this, this._tagRep)
		this.cache[partnerId] = v
	}
	return v
}

func (this *saleRep) GetValueGoods(partnerId, goodsId int) *sale.ValueGoods {
	var e *sale.ValueGoods = new(sale.ValueGoods)
	err := this.Connector.GetOrm().GetByQuery(e, `select * FROM gs_goods
			INNER JOIN gs_category c ON c.id = gs_goods.category_id WHERE gs_goods.id=?
			AND c.partner_id=?`, goodsId, partnerId)
	if err != nil {
		return nil
	}
	return e
}

func (this *saleRep) GetGoodsByIds(ids ...int) ([]*sale.ValueGoods, error) {
	//todo: partnerId
	var items []*sale.ValueGoods

	//todo:改成database/sql方式，不使用orm
	err := this.Connector.GetOrm().SelectByQuery(&items,
		`SELECT * FROM gs_goods WHERE id IN (`+ format.GetCategoryIdStr(ids)+`)`)

	return items, err
}

func (this *saleRep) SaveGoods(v *sale.ValueGoods) (int, error) {
	orm := this.Connector.GetOrm()
	if v.Id <= 0 {
		_, id, err := orm.Save(nil, v)
		return int(id), err
	} else {
		_, _, err := orm.Save(v.Id, v)
		return v.Id, err
	}
}

func (this *saleRep) GetPagedOnShelvesGoods(partnerId int, catIds []int, num int) (total int,e []*sale.ValueGoods) {
	var sql string
	if num <= 0 {
		num = 10
	}

	var catIdStr string = format.GetCategoryIdStr(catIds)
	sql = fmt.Sprintf(`SELECT * FROM gs_goods INNER JOIN gs_category ON gs_goods.category_id=gs_category.id
		WHERE partner_id=%d AND gs_category.id IN (%s) AND on_shelves=1 LIMIT 0,%d`, partnerId, catIdStr, num)

	this.Connector.ExecScalar(fmt.Sprintf(`SELECT COUNT(0) FROM gs_goods INNER JOIN gs_category ON gs_goods.category_id=gs_category.id
		WHERE partner_id=%d AND gs_category.id IN (%s) AND on_shelves=1`, partnerId, catIdStr),&total)

	e = []*sale.ValueGoods{}
	this.Connector.GetOrm().SelectByQuery(&e, sql)

	return total,e
}

func (this *saleRep) DeleteGoods(partnerId, goodsId int) error {
	_, _, err := this.Connector.Exec(`
		DELETE f,f2 FROM gs_goods AS f
		INNER JOIN gs_category AS c ON f.category_id=c.id
		INNER JOIN gs_goodsprop as f2 ON f2.id=f.id
		WHERE f.id=? AND c.partner_id=?`, goodsId, partnerId)
	return err
}

//获取食物数量
//todo: 还未使用
func (this *saleRep) FoodItemsCount(partnerId, cid int) (count int) {
	this.Connector.QueryRow(`
		SELECT COUNT(0) FROM gs_goods f
	INNER JOIN gs_category c ON f.category_id = c.id
	 where c.partner_id = ?
	AND (cid == -1 OR cid = ?)
	`, func(r *sql.Row) {
		r.Scan(count)
	}, partnerId, cid)
	return count
}

func (this *saleRep) SaveCategory(v *sale.ValueCategory) (int, error) {
	orm := this.Connector.GetOrm()
	if v.Id <= 0 {
		_, _, err := orm.Save(nil, v)
		if err == nil {
			this.Connector.ExecScalar(`SELECT MAX(id) FROM gs_category`, &v.Id)
		}
		return v.Id, err
	} else {
		_, _, err := orm.Save(v.Id, v)
		return v.Id, err
	}
}

func (this *saleRep) DeleteCategory(partnerId, id int) error {
	//删除子类
	_, _, err := this.Connector.Exec("DELETE FROM gs_category WHERE partner_id=? AND parent_id=?",
		partnerId, id)

	//删除分类
	_, _, err = this.Connector.Exec("DELETE FROM gs_category WHERE partner_id=? AND id=?",
		partnerId, id)

	//清理项
	this.Connector.Exec(`DELETE FROM gs_goods WHERE Cid NOT IN
		(SELECT Id FROM gs_category WHERE partner_id=?)`, partnerId)

	return err
}

func (this *saleRep) GetCategory(partnerId, id int) *sale.ValueCategory {
	var e *sale.ValueCategory = new(sale.ValueCategory)
	err := this.Connector.GetOrm().Get(id, e)
	if err != nil {
		log.PrintErr(err)
		return nil
	}

	if e.PartnerId != partnerId {
		return nil
	}

	return e
}

func (this *saleRep) GetCategories(partnerId int) []*sale.ValueCategory {
	var e []*sale.ValueCategory = []*sale.ValueCategory{}
	err := this.Connector.GetOrm().Select(&e, "partner_id=? ORDER BY id ASC", partnerId)
	if err != nil {
		log.PrintErr(err)
	}
	return e
}

// 获取与栏目相关的栏目
func (this *saleRep) GetRelationCategories(partnerId, categoryId int) []*sale.ValueCategory {
	var all []*sale.ValueCategory = this.GetCategories(partnerId)
	var newArr []*sale.ValueCategory = []*sale.ValueCategory{}
	var isMatch bool
	var pid int
	var l int = len(all)

	for i := 0; i < l; i++ {
		if !isMatch && all[i].Id == categoryId {
			isMatch = true
			pid = all[i].ParentId
			newArr = append(newArr, all[i])
			i = -1
		} else {
			if all[i].Id == pid {
				newArr = append(newArr, all[i])
				pid = all[i].ParentId
				i = -1
				if pid == 0 {
					break
				}
			}
		}
	}
	return newArr
}



// 获取子栏目
func (this *saleRep) GetChildCategories(partnerId,categoryId int)[]*sale.ValueCategory{
	var all []*sale.ValueCategory = this.GetCategories(partnerId)
	var newArr []*sale.ValueCategory = []*sale.ValueCategory{}

	var cdt iterator.Condition = func(v,v1 interface{})bool{
		return v1.(*sale.ValueCategory).ParentId == v.(*sale.ValueCategory).Id
	}
	var start iterator.WalkFunc = func(v interface{},level int){
		c := v.(*sale.ValueCategory)
		if c.Id != categoryId {
			newArr = append(newArr, c)
		}
	}

	var arr []interface{} = make([]interface{},len(all))
	for i,_:= range arr{
		arr[i] = all[i]
	}

	iterator.Walk(arr,&sale.ValueCategory{Id:categoryId},cdt,start,nil,1)

	return newArr
}

// 保存快照
func (this *saleRep) SaveSnapshot(v *sale.GoodsSnapshot) (int, error) {
	var id int
	_, _, err := this.Connector.GetOrm().Save(nil, v)
	if err == nil {
		err = this.Connector.ExecScalar(`SELECT MAX(id) FROM gs_snapshot where goods_id=?`, &id, v.GoodsId)
	}

	return id, err
}

// 获取最新的商品快照
func (this *saleRep) GetLatestGoodsSnapshot(goodsId int) *sale.GoodsSnapshot {
	var e *sale.GoodsSnapshot = new(sale.GoodsSnapshot)
	err := this.Connector.GetOrm().GetBy(e, "goods_id=? ORDER BY id DESC", goodsId)
	if err != nil {
		log.PrintErr(err)
		e = nil
	}
	return e
}

// 获取指定的商品快照
func (this *saleRep) GetGoodsSnapshot(id int) *sale.GoodsSnapshot {
	var e *sale.GoodsSnapshot = new(sale.GoodsSnapshot)
	err := this.Connector.GetOrm().Get(e, id)
	if err != nil {
		log.PrintErr(err)
		e = nil
	}
	return e
}

// 根据Key获取商品快照
func (this *saleRep) GetGoodsSnapshotByKey(key string) *sale.GoodsSnapshot {
	var e *sale.GoodsSnapshot = new(sale.GoodsSnapshot)
	err := this.Connector.GetOrm().GetBy(e, "key=?", key)
	if err != nil {
		log.PrintErr(err)
		e = nil
	}
	return e
}
