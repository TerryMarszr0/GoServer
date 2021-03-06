package tcp

import (
	"common"
	"common/net/meta"
	"encoding/binary"
	"gamelog"
	"generate_out/rpc/enum"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPServer struct {
	Addr       string //"ip:port"，ip可缺省
	MaxConnNum int
	autoConnId uint32
	connmap    map[uint32]*TCPConn
	listener   net.Listener
	mutexConns sync.Mutex
	wgLn       sync.WaitGroup
	wgConns    sync.WaitGroup
}

func NewTcpServer(addr string, maxconn int) {
	svr := new(TCPServer)
	svr.Addr = addr
	svr.MaxConnNum = maxconn
	svr.init()
	svr.run()
	svr.Close()
}
func (self *TCPServer) init() bool {
	ln, err := net.Listen("tcp", self.Addr)
	if err != nil {
		gamelog.Error("TCPServer Init failed  error :%s", err.Error())
		return false
	}
	self.listener = ln
	self.connmap = make(map[uint32]*TCPConn)
	return true
}
func (self *TCPServer) run() {
	self.wgLn.Add(1)
	defer self.wgLn.Done()
	var tempDelay time.Duration
	for {
		conn, err := self.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if tempDelay > time.Second {
					tempDelay = time.Second
				}
				gamelog.Error("accept error: %s retrying in %d", err.Error(), tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			gamelog.Error("accept error: %s", err.Error())
			return
		}
		tempDelay = 0
		go self._HandleAcceptConn(conn)
	}
}
func (self *TCPServer) _HandleAcceptConn(conn net.Conn) {
	buf := make([]byte, 2+4) //Notice：收第一个包，客户端上报connId
	if _, err := io.ReadFull(conn, buf); err != nil {
		conn.Close()
		gamelog.Error("RecvFirstPack: %s", err.Error())
		return
	}
	connId := binary.LittleEndian.Uint32(buf[2:])
	gamelog.Info("_HandleAcceptConn: %d", connId)

	if connId > 0 {
		self._ResetOldConn(conn, connId)
	} else {
		self._AddNewConn(conn)
	}
}
func (self *TCPServer) _AddNewConn(conn net.Conn) {
	if len(self.connmap) >= self.MaxConnNum {
		conn.Close()
		gamelog.Error("too many connections(%d/%d)", len(self.connmap), self.MaxConnNum)
		return
	}

	connId := atomic.AddUint32(&self.autoConnId, 1)

	self.wgConns.Add(1)
	tcpConn := newTCPConn(conn,
		func(this *TCPConn) {
			self.mutexConns.Lock()
			delete(self.connmap, connId)
			self.mutexConns.Unlock()
			gamelog.Info("Disconnect: UserPtr:%v, DelConnId: %d, ConnCnt: %d", this.UserPtr, connId, len(self.connmap))
			self.wgConns.Done()
		})
	self.mutexConns.Lock()
	self.connmap[connId] = tcpConn
	self.mutexConns.Unlock()
	gamelog.Info("Connect From: %s, AddConnId: %d, ConnCnt: %d", conn.RemoteAddr().String(), connId, len(self.connmap))

	// 通知client，连接被接收，下发connId、密钥等
	acceptMsg := common.NewNetPackCap(32)
	acceptMsg.SetOpCode(enum.Rpc_svr_accept)
	acceptMsg.WriteUInt32(connId)
	tcpConn.WriteMsg(acceptMsg)
	go tcpConn.readRoutine()
	tcpConn.writeRoutine()
}
func (self *TCPServer) _ResetOldConn(newconn net.Conn, oldId uint32) {
	self.mutexConns.Lock()
	oldconn, ok := self.connmap[oldId]
	self.mutexConns.Unlock()
	if oldconn != nil && ok {
		if oldconn.isClose {
			gamelog.Info("_ResetOldConn isClose: %d", oldId)
			oldconn.ResetConn(newconn)
			go oldconn.readRoutine()
			oldconn.writeRoutine()
		} else {
			gamelog.Info("_ResetOldConn isOpen: %d", oldId)
			// newconn.Close()
			self._AddNewConn(newconn)
		}
	} else { //服务器重启
		gamelog.Info("_ResetOldConn to _AddNewConn: %d", oldId)
		self._AddNewConn(newconn)
	}
}
func (self *TCPServer) Close() {
	self.listener.Close()
	self.wgLn.Wait()

	self.mutexConns.Lock()
	for _, conn := range self.connmap {
		conn.Close()
	}
	self.connmap = nil
	self.mutexConns.Unlock()

	self.wgConns.Wait()
	gamelog.Info("server been closed!!")
}

//////////////////////////////////////////////////////////////////////
//! 模块注册
type TRegConn struct {
	*meta.Meta
	Conn *TCPConn
}

var g_reg_conn_map sync.Map

func DoRegistToSvr(req, ack *common.NetPack, conn *TCPConn) {
	ptr := new(meta.Meta)
	ptr.BufToData(req)

	g_reg_conn_map.Store(common.KeyPair{ptr.Module, ptr.SvrID}, TRegConn{ptr, conn})
	meta.AddMeta(ptr)

	cb := conn.onNetClose
	conn.onNetClose = func(this *TCPConn) {
		if pConn := FindRegModuleConn(ptr.Module, ptr.SvrID); pConn != nil && pConn.isClose {
			gamelog.Info("Delete Regist Svr: {%s %d}", ptr.Module, ptr.SvrID)
			g_reg_conn_map.Delete(common.KeyPair{ptr.Module, ptr.SvrID})
		}
		if cb != nil {
			cb(this)
		}
	}
	gamelog.Info("DoRegistToSvr: {%s %d}", ptr.Module, ptr.SvrID)
}

func FindRegModuleConn(module string, id int) *TCPConn {
	if v, ok := FindRegModule(module, id); ok {
		return v.Conn
	}
	return nil
}
func FindRegModule(module string, id int) (TRegConn, bool) {
	if v, ok := g_reg_conn_map.Load(common.KeyPair{module, id}); ok {
		return v.(TRegConn), true
	}
	gamelog.Error("FindRegModuleConn nil : (%s,%d) => %v", module, id, g_reg_conn_map)
	return TRegConn{}, false
}

func GetRegModuleIDs(module string) (ret []int) {
	g_reg_conn_map.Range(func(k, v interface{}) bool {
		if k.(common.KeyPair).Name == module {
			ret = append(ret, k.(common.KeyPair).ID)
		}
		return true
	})
	return
}
func ForeachRegModule(f func(v TRegConn)) {
	g_reg_conn_map.Range(func(k, v interface{}) bool {
		f(v.(TRegConn))
		return true
	})
}
