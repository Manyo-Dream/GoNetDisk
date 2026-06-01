package repository

import "gorm.io/gorm"

// TxManager 事务管理器，闭包自动管理 提交/回滚
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// Transaction 在事务中执行业务逻辑，返回 error 自动回滚，nil 自动提交
func (m *TxManager) Transaction(fn func(tx *gorm.DB) error) error {
	return m.db.Transaction(fn)
}
