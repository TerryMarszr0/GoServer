package api

import (
	"common"
	"netConfig"
	"tcp"
)

var (
	g_cache_game_conn = make(map[int]*tcp.TCPConn)
)

func SendToGame(svrId int, msg *common.NetPack) {
	conn, _ := g_cache_game_conn[svrId]
	if conn == nil || conn.IsClose() {
		conn = netConfig.GetTcpConn("game", svrId)
		g_cache_game_conn[svrId] = conn
	}
	conn.WriteMsg(msg)
}
