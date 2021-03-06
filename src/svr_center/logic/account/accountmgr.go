package account

import (
	"dbmgo"
	"sync"
	"time"

	"gopkg.in/mgo.v2/bson"
)

var (
	g_name_cache sync.Map //map[string]*TAccount
	g_aid_cache  sync.Map //map[uint32]*TAccount
)

func InitDB() {
	//只载入一个月内登录过的
	var list []TAccount
	dbmgo.FindAll("Account", bson.M{"logintime": bson.M{"$gt": time.Now().Unix() - 30*24*3600}}, &list)
	for i := 0; i < len(list); i++ {
		AddCache(&list[i])
	}
	println("load active account form db: ", len(list))
}
func AddNewAccount(name, password string) *TAccount {
	account := &TAccount{
		Name:       name,
		Password:   password,
		CreateTime: time.Now().Unix(),
	}
	if dbmgo.Find("Account", "name", name, account) {
		return nil
	}
	account.AccountID = dbmgo.GetNextIncId("AccountId")

	if dbmgo.InsertSync("Account", account) {
		AddCache(account)
		return account
	}
	return nil
}
func GetAccountByName(name string) *TAccount {
	if v, ok := g_name_cache.Load(name); ok {
		return v.(*TAccount)
	} else {
		account := new(TAccount)
		if ok := dbmgo.Find("Account", "name", name, account); ok {
			AddCache(account)
			return account
		}
	}
	return nil
}
func GetAccountById(accountId uint32) *TAccount {
	if v, ok := g_aid_cache.Load(accountId); ok {
		return v.(*TAccount)
	} else {
		account := new(TAccount)
		if ok := dbmgo.Find("Account", "_id", accountId, account); ok {
			AddCache(account)
			return account
		}
	}
	return nil
}
func ResetPassword(name, password, newpassword string) bool {
	if account := GetAccountByName(name); account != nil {
		if account.Password == password {
			account.Password = newpassword
			dbmgo.UpdateToDB("Account", bson.M{"_id": account.AccountID}, bson.M{"$set": bson.M{
				"password": newpassword}})
			return true
		}
	}
	return false
}

// -------------------------------------
//! 辅助函数
func AddCache(account *TAccount) {
	g_name_cache.Store(account.Name, account)
	g_aid_cache.Store(account.AccountID, account)
}
func DelCache(account *TAccount) {
	g_name_cache.Delete(account.Name)
	g_aid_cache.Delete(account.AccountID)
}
