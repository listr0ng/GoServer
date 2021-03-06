/***********************************************************************
* @ http rpc
* @ brief
	1、system rpc：将原生http的参数统一转换为NetPack
	2、player rpc：在system rpc基础之上，加了层find player逻辑，若找不到不处理

* @ Notic
	1、http的消息处理，是另开goroutine调用的，所以函数中可阻塞；tcp就不行了

	2、正因为每条消息都是另开goroutine，若玩家连续发多条消息，服务器就是并发处理了，存在竞态……client确保应答式通信

	3、http服务器自带多线程环境，写业务代码危险多了，须十分注意共享数据的保护
		· 全局变量
		· 队伍数据
		· 聊天记录（只要不是独属自己的数据，都得加保护~囧）

* @ http消息回调
	http._doRegistToSvr(0x8c8d60, 0xc042160380, 0xc0421c6000)
		D:/soulnet/GoServer/src/http/http_server.go:38 +0x3b
	net/http.HandlerFunc.ServeHTTP(0x76e638, 0x8c8d60, 0xc042160380, 0xc0421c6000)
		C:/Go/src/net/http/server.go:1918 +0x4b
	net/http.(*ServeMux).ServeHTTP(0x8fd800, 0x8c8d60, 0xc042160380, 0xc0421c6000)
		C:/Go/src/net/http/server.go:2254 +0x137
	net/http.serverHandler.ServeHTTP(0xc042158410, 0x8c8d60, 0xc042160380, 0xc0421c6000)
		C:/Go/src/net/http/server.go:2619 +0xbb
	net/http.(*conn).serve(0xc0421c0000, 0x8c91e0, 0xc0420343c0)
		C:/Go/src/net/http/server.go:1801 +0x724
	created by net/http.(*Server).Serve
		C:/Go/src/net/http/server.go:2720 +0x28f

* @ author zhoumf
* @ date 2017-8-10
***********************************************************************/
package http

import (
	"common"
	"encoding/binary"
	"gamelog"
	"generate_out/rpc/enum"
	"net/http"
)

const (
	Client_ReLogin_Flag = 0xFFFFFFFF
)

var (
	G_HandleFunc       [enum.RpcEnumCnt]func(req, ack *common.NetPack)
	G_PlayerHandleFunc [enum.RpcEnumCnt]func(req, ack *common.NetPack, p interface{})

	//! 需要主动发给玩家的数据，每回通信时捎带过去
	G_Before_Recv_Player func(uint32) interface{}
	G_After_Recv_Player  func(interface{}, *common.NetPack)
)

// ------------------------------------------------------------
//! system rpc
func CallRpc(addr string, rid uint16, sendFun, recvFun func(*common.NetPack)) {
	req := common.NewNetPackCap(64)
	req.SetOpCode(rid)
	sendFun(req)
	if buf := PostReq(addr+"client_rpc", req.Data()); buf != nil && recvFun != nil {
		ack := common.NewNetPack(common.Decompress(buf))
		recvFun(ack)
	}
	req.Free()
}
func RegHandleRpc() { http.HandleFunc("/client_rpc", _HandleRpc) }
func _HandleRpc(w http.ResponseWriter, r *http.Request) {
	//! 接收信息
	req := common.NewNetPackLen(int(r.ContentLength))
	r.Body.Read(req.Data())

	//! 创建回复
	ack := common.NewNetPackCap(128)
	msgId := req.GetOpCode()
	defer ack.Free()
	//defer func() {//库已经有recover了，见net/http/server.go:1918
	//	if r := recover(); r != nil {
	//		gamelog.Error("recover msgId:%d\n%v: %s", msgId, r, debug.Stack())
	//	}
	//	ack.Free()
	//}()

	gamelog.Debug("HttpMsg:%d, len:%d", msgId, req.Size())

	if handler := G_HandleFunc[msgId]; handler != nil {
		handler(req, ack)
		common.CompressInto(ack.Data(), w)
	} else {
		println("\n===> Http HandleRpc:", msgId, "Not Regist!!!")
	}
}

// ------------------------------------------------------------
//! player rpc
type PlayerRpc struct {
	Url      string
	PlayerId uint32
}

func NewPlayerRpc(addr string, pid uint32) *PlayerRpc {
	return &PlayerRpc{addr + "player_rpc", pid}
}
func (self *PlayerRpc) CallRpc(rid uint16, sendFun, recvFun func(*common.NetPack)) {
	req := common.NewNetPackCap(64)
	req.SetOpCode(rid)
	req.SetReqIdx(self.PlayerId)
	sendFun(req)
	if buf := PostReq(self.Url, req.Data()); buf != nil {
		ack := common.NewNetPack(common.Decompress(buf))
		if recvFun != nil {
			recvFun(ack)
		}
		_RecvHttpSvrData(ack) //服务器主动下发的数据
	}
	req.Free()
}
func _RecvHttpSvrData(buf *common.NetPack) {
	//对应于 http_to_client.go
}
func RegHandlePlayerRpc() { http.HandleFunc("/player_rpc", _HandlePlayerRpc) }
func _HandlePlayerRpc(w http.ResponseWriter, r *http.Request) {
	//! 接收信息
	req := common.NewNetPackLen(int(r.ContentLength))
	r.Body.Read(req.Data())

	//! 创建回复
	ack := common.NewNetPackCap(128)
	msgId := req.GetOpCode()
	pid := req.GetReqIdx()
	defer ack.Free()
	//defer func() {//库已经有recover了，见net/http/server.go:1918
	//	if r := recover(); r != nil {
	//		gamelog.Error("recover msgId:%d\n%v: %s", msgId, r, debug.Stack())
	//	}
	//	ack.Free()
	//}()
	//FIXME: 验证消息安全性，防改包
	//FIXME: http通信中途安全性不够，能修改client net pack里的pid，进而操作别人数据
	//FIXME: 账号服登录验证后下发给client的token，client应该保留，附在每个HttpReq里，防止恶意窜改他人数据

	if msgId != enum.Rpc_game_heart_beat {
		gamelog.Debug("HttpMsg:%d, len:%d, playerId:%d", msgId, req.Size(), pid)
	}
	if handler := G_PlayerHandleFunc[msgId]; handler != nil {
		var player interface{}
		if G_Before_Recv_Player != nil {
			player = G_Before_Recv_Player(pid)
		}
		if player == nil {
			gamelog.Debug("===> pid:%d isn't in memcache, please relogin", pid)
			flag := make([]byte, 4) //重登录标记
			binary.LittleEndian.PutUint32(flag, Client_ReLogin_Flag)
			w.Write(flag)
			return
		}
		handler(req, ack, player)

		if G_After_Recv_Player != nil && player != nil {
			G_After_Recv_Player(player, ack)
		}
		common.CompressInto(ack.Data(), w)
	} else {
		println("\n===> Http HandlePlayerRpc:", msgId, "Not Regist!!!")
	}
}
