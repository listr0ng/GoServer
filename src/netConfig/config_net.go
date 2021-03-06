/***********************************************************************
* @ 多进程服务器架构
* @ brief
	1、游戏服，可多个GameSvr对应同一区服玩家
	2、战斗服，可动态扩展；GameSvr将玩家数据通过Cross转至某Battle
	3、唯一的Zookeeper，其它进程需在Zookeeper注册，管理后台所有进程

* @ reconnect
	1、【1-1】关系中的"client"重启：game每次均会连接battle
	2、【1-1】关系中的"server"重启：cross(tcp)重启，game的TCPClient.connectRoutine能检查到失败，循环重连
	3、【1-N】关系中的"N"重启：game每次均会去sdk注册
	4、【1-N】关系中的"1"重启：http_server.go会本地存储注册地址，重启时载入

* @ author zhoumf
* @ date 2016-8-11
***********************************************************************/
package netConfig

import (
	"common"
	"common/net/meta"
	"fmt"
	"gamelog"
	"http"
	"sync"
	"tcp"
)

var (
	G_Local_Meta   *meta.Meta
	G_Client_Conns sync.Map //= make(map[common.KeyPair]*tcp.TCPClient) //本模块，对其它模块的tcp连接
)

func RunNetSvr() {
	//1、找到当前的配置信息
	common.Assert(G_Local_Meta != nil)

	//2、连接/注册其它模块
	if nil == meta.GetMeta("zookeeper", 0) { //没有zookeeper节点，才依赖配置，否则依赖zookeeper的通知
		for _, destModule := range G_Local_Meta.ConnectLst {
			meta.G_SvrNets.Range(func(k, v interface{}) bool {
				destCfg := v.(*meta.Meta)
				if destCfg.Module == destModule {
					if destCfg.HttpPort > 0 {
						http.RegistToSvr(
							http.Addr(destCfg.IP, destCfg.HttpPort),
							G_Local_Meta)
					} else if destCfg.TcpPort > 0 {
						client := new(tcp.TCPClient)
						client.ConnectToSvr(
							tcp.Addr(destCfg.IP, destCfg.TcpPort),
							G_Local_Meta)
						//Notice：client.ConnectToSvr是异步过程，这里返回的client.TcpConn还是空指针，不能保存*tcp.TCPConn
						G_Client_Conns.Store(common.KeyPair{destCfg.Module, destCfg.SvrID}, client)
					} else {
						fmt.Println(destCfg.Module + ": have none HttpPort|TcpPort!!!")
					}
				}
				return true
			})
		}
	}

	//3、开启本模块网络服务(Busy Loop)
	fmt.Printf("-------%s server start-------\n", G_Local_Meta.Module)
	if G_Local_Meta.HttpPort > 0 {
		http.NewHttpServer(fmt.Sprintf(":%d", G_Local_Meta.HttpPort))
	} else if G_Local_Meta.TcpPort > 0 {
		tcp.NewTcpServer(fmt.Sprintf(":%d", G_Local_Meta.TcpPort), G_Local_Meta.Maxconn)
	} else {
		fmt.Println(G_Local_Meta.Module + ": have none HttpPort|TcpPort!!!")
	}
}

//Notice：应用层cache住结果，避免每次都查找
func GetTcpConn(module string, svrId int) *tcp.TCPConn {
	if v, ok := G_Client_Conns.Load(common.KeyPair{module, svrId}); ok {
		ptr := v.(*tcp.TCPClient)
		return ptr.Conn
	}
	// cross(s) - game(c)
	return tcp.FindRegModule(module, svrId)
}
func GetHttpAddr(module string, svrId int) string {
	if pMeta := meta.GetMeta(module, svrId); pMeta != nil {
		return http.Addr(pMeta.IP, pMeta.HttpPort)
	}
	gamelog.Error("GetHttpAddr nil : (%s,%d) %d", module, svrId)
	return ""
}
