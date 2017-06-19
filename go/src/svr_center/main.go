package main

import (
	"common"
	"conf"
	"dbmgo"
	"gamelog"
	"netConfig"
	"strconv"

	"svr_center/logic/account"
)

func main() {
	//初始化日志系统
	gamelog.InitLogger("center")
	gamelog.SetLevel(0)

	//设置mongodb的服务器地址
	dbmgo.Init(conf.AccountDbAddr, conf.AccountDbName)

	//开启控制台窗口，可以接受一些调试命令
	common.StartConsole()
	common.RegConsoleCmd("setloglevel", HandCmd_SetLogLevel)

	InitConf()

	account.G_AccountMgr.Init()

	gamelog.Warn("----Center Server Start-----")
	if netConfig.CreateNetSvr("center", 0) == false {
		gamelog.Error("----Center NetSvr Failed-----")
	}
}
func HandCmd_SetLogLevel(args []string) bool {
	if len(args) < 2 {
		gamelog.Error("Lack of param")
		return false
	}
	level, err := strconv.Atoi(args[1])
	if err != nil {
		gamelog.Error("HandCmd_SetLogLevel Error : Invalid param :%s", args[1])
		return false
	}
	gamelog.SetLevel(level)
	return true
}
func InitConf() {
	common.G_Csv_Map = map[string]interface{}{
		"conf_net": &netConfig.G_SvrNetCfg,
		"rpc":      &common.G_RpcCsv,
	}
	common.LoadAllCsv()

	netConfig.RegHttpSystemHandler(map[string]netConfig.HttpHandle{
		//! Gamesvr
		"rpc_center_login_game_success": account.Handle_Login_Game_Success,
	})
	netConfig.RegHttpPlayerHandler(map[string]netConfig.HttpPlayerHandle{
		//! Client
		"rpc_center_reg_account":            account.Rpc_Reg_Account,
		"rpc_center_change_password":        account.Rpc_Change_Password,
		"rpc_center_get_gamesvr_lst":        account.Rpc_GetGameSvr_Lst,
		"rpc_center_get_gamesvr_last_login": account.Rpc_GetGameSvr_LastLogin,
		"rpc_center_login_gamesvr":          account.Rpc_Login_GameSvr,
	})
}
