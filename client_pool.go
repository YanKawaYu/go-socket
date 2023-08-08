package gosocket

import "sync"

var clientPool *ClientPool
var readWriteLock *sync.RWMutex

// ClientPool is used to store all the connections on the server
// It follows singleton pattern which means there should be only one instance
// Please use GetClientPool to get the instance
//
// Since the pool will be accessed by different threads, it is required to use sync.RWMutex to avoid conflict
type ClientPool struct {
	onlineClientMap map[int64]*MessageHandler
}

func init() {
	clientPool = &ClientPool{
		onlineClientMap: make(map[int64]*MessageHandler),
	}
	readWriteLock = new(sync.RWMutex)
}

// GetClientPool is the only way to get the single ClientPool instance
// 获取连接池单例
func GetClientPool() *ClientPool {
	return clientPool
}

// SetClientByUid is used to put the handler into the map and mark the user is online
// 将用户连接加入连接池中，标记用户在线
func (clientPool *ClientPool) SetClientByUid(handler *MessageHandler, uid int64) {
	readWriteLock.Lock()
	clientPool.onlineClientMap[uid] = handler
	readWriteLock.Unlock()
}

// RemoveClientByUid is used to remove the corresponding handler and mark the user is offline
// 将用户从连接池中删除，标记用户下线
func (clientPool *ClientPool) RemoveClientByUid(uid int64) {
	readWriteLock.Lock()
	delete(clientPool.onlineClientMap, uid)
	readWriteLock.Unlock()
}

// GetClientByUid is used to get the corresponding handler and check whether the user is online
// 获取用户对应的连接
func (clientPool *ClientPool) GetClientByUid(uid int64) *MessageHandler {
	readWriteLock.RLock()
	handler := clientPool.onlineClientMap[uid]
	readWriteLock.RUnlock()
	return handler
}
