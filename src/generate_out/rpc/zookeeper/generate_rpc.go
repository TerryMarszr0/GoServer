
// Generated by GoServer/src/generat
// Don't edit !
package rpc
import (
	"common/net/register"
	"generate_out/rpc/enum"
	
	
	"zookeeper/logic"
)
func init() {
	register.RegTcpRpc(map[uint16]register.TcpRpc{
		
		enum.Rpc_zoo_register: logic.Rpc_zoo_register,
	})
	register.RegHttpRpc(map[uint16]register.HttpRpc{
		
	})
	register.RegHttpPlayerRpc(map[uint16]register.HttpPlayerRpc{
		
	})
	register.RegHttpHandler(map[string]register.HttpHandle{
		
	})
}
