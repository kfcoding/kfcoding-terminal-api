package service

import (
	"github.com/kfcoding-terminal-controller/service/common"
	"log"
	"github.com/coreos/etcd/clientv3"
	"strings"
	"context"
	"github.com/kfcoding-terminal-controller/config"
	"path"
)

type EtcdService struct {
	myEtcdClient *common.MyEtcdClient
	onDelete     OnCloseCallback
}

func GetEtcdService(myEtcdClient *common.MyEtcdClient) *EtcdService {
	return &EtcdService{
		myEtcdClient: myEtcdClient,
	}
}

func (service *EtcdService) SetOnDeleteCallback(callback OnCloseCallback) {
	service.onDelete = callback
}

func (service *EtcdService) PutSessionId(sessionId string) error {

	var resp *clientv3.LeaseGrantResponse
	var err error
	if resp, err = service.myEtcdClient.EctdClientV3.Grant(context.TODO(), int64(config.KeeperTTL)); err != nil {
		log.Println("PutSessionId error: ", err)
		return err
	}

	key := path.Join(config.KeeperPrefix, config.Version, sessionId)
	if _, err = service.myEtcdClient.EctdClientV3.Put(context.TODO(), key, "", clientv3.WithLease(resp.ID)); nil != err {
		log.Println("PutSessionId error: ", err)
		return err
	}
	log.Println("PutSessionId ok")

	return nil
}

func (service *EtcdService) DeleteSessionId(sessionId string) {
	log.Print("DeleteSessionId ok: ", sessionId)
	key := path.Join(config.KeeperPrefix, config.Version, sessionId)
	service.myEtcdClient.Delete(key)
}

func (service *EtcdService) WatchSessionId(prefix string) {
	log.Println("Start Etcd Watcher")

	rch := service.myEtcdClient.EctdClientV3.Watch(context.Background(), prefix, clientv3.WithPrefix())

	for wresp := range rch {
		for _, ev := range wresp.Events {
			// /kfcoding/v1/2/terminal-3jf9afndks
			switch ev.Type {
			case 1: //DELETE
				log.Print("listen etcd delete: ", string(ev.Kv.Key))
				strs := strings.Split(string(ev.Kv.Key), "/")
				service.onDelete(strs[len(strs)-1], config.SourceEtcd)
			}
		}
	}
}
