package repo

import (
	"gorm.io/gorm"
	"lark/domain/do"
	"lark/domain/po"
	"lark/pkg/common/xmysql"
	"lark/pkg/entity"
	"lark/pkg/proto/pb_chat_member"
)

type ChatMemberRepository interface {
	TxCreate(tx *gorm.DB, chatMember *po.ChatMember) (err error)
	TxCreateMultiple(tx *gorm.DB, users []*po.ChatMember) (err error)
	ChatIdList(w *entity.MysqlWhere) (list []int64, err error)
	ChatMemberStatusList(w *entity.MysqlWhere) (list []*do.ChatMemberStatus, err error)
	PushMemberList(w *entity.MysqlWhere) (list []*pb_chat_member.PushMember, err error)
	PushMember(w *entity.MysqlWhere) (conf *pb_chat_member.PushMember, err error)
	ChatMember(w *entity.MysqlWhere) (member *pb_chat_member.ChatMemberInfo, err error)
	ChatMemberCount(w *entity.MysqlWhere) (count int64, err error)
	TxChatMemberList(tx *gorm.DB, w *entity.MysqlWhere) (members []*pb_chat_member.ChatMemberInfo, err error)
	UpdateChatMember(u *entity.MysqlUpdate) (err error)
	TxUpdateChatMember(tx *gorm.DB, u *entity.MysqlUpdate) (err error)
	TxQuitChatMember(tx *gorm.DB, u *entity.MysqlUpdate) (rowsAffected int64, err error)
	ChatMemberBasicInfoList(w *entity.MysqlWhere) (list []*pb_chat_member.ChatMemberBasicInfo, err error)
	GroupChatBasicInfoList(w *entity.MysqlWhere) (list []*pb_chat_member.GroupChatBasicInfo, err error)
	GroupChatMemberInfoList(w *entity.MysqlWhere) (list []*pb_chat_member.GroupChatMemberInfo, err error)
}

type chatMemberRepository struct {
}

func NewChatMemberRepository() ChatMemberRepository {
	return &chatMemberRepository{}
}

func (r *chatMemberRepository) TxCreate(tx *gorm.DB, chatMember *po.ChatMember) (err error) {
	err = tx.Create(chatMember).Error
	return
}

func (r *chatMemberRepository) TxCreateMultiple(tx *gorm.DB, users []*po.ChatMember) (err error) {
	err = tx.Create(users).Error
	return
}

func (r *chatMemberRepository) ChatIdList(w *entity.MysqlWhere) (list []int64, err error) {
	list = make([]int64, 0)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Where(w.Query, w.Args...).Limit(w.Limit).Pluck("chat_id", &list).Error
	return
}

func (r *chatMemberRepository) ChatMemberStatusList(w *entity.MysqlWhere) (list []*do.ChatMemberStatus, err error) {
	list = make([]*do.ChatMemberStatus, 0)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).
		Select("chat_id,status").
		Where(w.Query, w.Args...).
		Order(w.Sort).
		Limit(w.Limit).
		Find(&list).Error
	return
}

//func (r *chatMemberRepository) ChatMemberList(w *entity.MysqlWhere) (list []*do.ChatMemberInfo, err error) {
//	list = make([]*do.ChatMemberInfo, 0)
//	db := xmysql.GetDB()
//	err = db.Model(po.ChatMember{}).
//		Select("chat_id,uid,status,server_id").
//		Where(w.Query, w.Args...).
//		Order(w.Sort).
//		Limit(w.Limit).Find(&list).Error
//	return
//}

func (r *chatMemberRepository) PushMemberList(w *entity.MysqlWhere) (list []*pb_chat_member.PushMember, err error) {
	list = make([]*pb_chat_member.PushMember, 0)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Select("uid,status,server_id").Where(w.Query, w.Args...).Find(&list).Error
	return
}

func (r *chatMemberRepository) PushMember(w *entity.MysqlWhere) (conf *pb_chat_member.PushMember, err error) {
	conf = new(pb_chat_member.PushMember)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Select("chat_id,uid,status,server_id").Where(w.Query, w.Args...).Find(&conf).Error
	return
}

func (r *chatMemberRepository) ChatMember(w *entity.MysqlWhere) (member *pb_chat_member.ChatMemberInfo, err error) {
	member = new(pb_chat_member.ChatMemberInfo)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Select("chat_id,chat_type,uid,alias,member_avatar_key,role_id,status").Where(w.Query, w.Args...).Find(&member).Error
	return
}

func (r *chatMemberRepository) ChatMemberCount(w *entity.MysqlWhere) (count int64, err error) {
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Where(w.Query, w.Args...).Count(&count).Error
	return
}

func (r *chatMemberRepository) TxChatMemberList(tx *gorm.DB, w *entity.MysqlWhere) (members []*pb_chat_member.ChatMemberInfo, err error) {
	members = make([]*pb_chat_member.ChatMemberInfo, 0)
	err = tx.Model(po.ChatMember{}).
		Select("chat_id,chat_type,uid,alias,member_avatar_key,role_id").
		Where(w.Query, w.Args...).
		Limit(w.Limit).
		Find(&members).Error
	return
}

func (r *chatMemberRepository) UpdateChatMember(u *entity.MysqlUpdate) (err error) {
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Where(u.Query, u.Args...).Updates(u.Values).Error
	return
}

func (r *chatMemberRepository) TxUpdateChatMember(tx *gorm.DB, u *entity.MysqlUpdate) (err error) {
	err = tx.Model(po.ChatMember{}).Where(u.Query, u.Args...).Updates(u.Values).Error
	return
}

func (r *chatMemberRepository) TxQuitChatMember(tx *gorm.DB, u *entity.MysqlUpdate) (rowsAffected int64, err error) {
	result := tx.Model(po.ChatMember{}).Where(u.Query, u.Args...).Updates(u.Values)
	err = result.Error
	rowsAffected = result.RowsAffected
	return
}

func (r *chatMemberRepository) ChatMemberBasicInfoList(w *entity.MysqlWhere) (list []*pb_chat_member.ChatMemberBasicInfo, err error) {
	list = make([]*pb_chat_member.ChatMemberBasicInfo, 0)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Select("uid,alias,remark,member_avatar_key,status").
		Where(w.Query, w.Args...).
		Order(w.Sort).
		Limit(w.Limit).
		Find(&list).Error
	return
}

func (r *chatMemberRepository) GroupChatMemberInfoList(w *entity.MysqlWhere) (list []*pb_chat_member.GroupChatMemberInfo, err error) {
	list = make([]*pb_chat_member.GroupChatMemberInfo, 0)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Select("uid,alias,member_avatar_key,role_id,status").
		Where(w.Query, w.Args...).
		Order(w.Sort).
		Limit(w.Limit).
		Find(&list).Error
	return
}

func (r *chatMemberRepository) GroupChatBasicInfoList(w *entity.MysqlWhere) (list []*pb_chat_member.GroupChatBasicInfo, err error) {
	list = make([]*pb_chat_member.GroupChatBasicInfo, 0)
	db := xmysql.GetDB()
	err = db.Model(po.ChatMember{}).Select("chat_id,chat_name,remark,chat_avatar_key").
		Where(w.Query, w.Args...).
		Order(w.Sort).
		Limit(w.Limit).
		Find(&list).Error
	return
}
