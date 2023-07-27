package gotcp

import "sync"

var clientPool *ClientPool
var readWriteLock *sync.RWMutex

type ClientPool struct {
	onlineClientMap map[int64]*MessageHandler
}

func init()  {
	clientPool = &ClientPool{
		onlineClientMap: make(map[int64]*MessageHandler),
	}
	readWriteLock = new(sync.RWMutex)
}

//获取连接池单例
func GetClientPool() *ClientPool {
	return clientPool
}

//将用户连接加入连接池中，标记用户在线
func (clientPool *ClientPool) SetClientByUid(handler *MessageHandler, uid int64) {
	readWriteLock.Lock()
	clientPool.onlineClientMap[uid] = handler
	readWriteLock.Unlock()
}

//将用户从连接池中删除，标记用户下线
func (clientPool *ClientPool) RemoveClientByUid(uid int64) {
	readWriteLock.Lock()
	delete(clientPool.onlineClientMap, uid)
	readWriteLock.Unlock()
}

//获取用户对应的连接
func (clientPool *ClientPool) GetClientByUid(uid int64) *MessageHandler {
	readWriteLock.RLock()
	handler := clientPool.onlineClientMap[uid]
	readWriteLock.RUnlock()
	return handler
}