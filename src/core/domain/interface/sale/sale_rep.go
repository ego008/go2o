/**
 * Copyright 2014 @ S1N1 Team.
 * name :
 * author : jarryliu
 * date : 2013-12-08 10:45
 * description :
 * history :
 */

package sale

// 销售仓库
type ISaleRep interface {
	GetSale(partnerId int) ISale

	GetValueGoods(partnerId, goodsId int) *ValueGoods

	GetGoodsByIds(ids ...int) ([]*ValueGoods, error)

	SaveGoods(*ValueGoods) (int, error)

	// 获取在货架上的商品
	GetPagedOnShelvesGoods(partnerId int, catIds []int, num int)(total int,goods []*ValueGoods)

	DeleteGoods(partnerId, goodsId int) error

	SaveCategory(*ValueCategory) (int, error)

	DeleteCategory(partnerId, id int) error

	GetCategory(partnerId, id int) *ValueCategory

	GetCategories(partnerId int) []*ValueCategory

	// 获取与栏目相关的栏目
	GetRelationCategories(partnerId, categoryId int) []*ValueCategory

	// 获取子栏目
	GetChildCategories(partnerId,categoryId int)[]*ValueCategory

	// 保存快照
	SaveSnapshot(*GoodsSnapshot) (int, error)

	// 获取最新的商品快照
	GetLatestGoodsSnapshot(goodsId int) *GoodsSnapshot

	// 获取指定的商品快照
	GetGoodsSnapshot(id int) *GoodsSnapshot

	// 根据Key获取商品快照
	GetGoodsSnapshotByKey(key string) *GoodsSnapshot
}
