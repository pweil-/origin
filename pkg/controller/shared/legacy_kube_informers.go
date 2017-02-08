package shared

import (
	"reflect"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

type SecretInformer interface {
	Informer() cache.SharedIndexInformer
	Indexer() cache.Indexer
	// Lister is not implemented for SecretInformer
	// TODO: Remove this after 1.6 rebase.
	Lister() interface{}
}

type secretInformer struct {
	*sharedInformerFactory
}

func (f *secretInformer) Informer() cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	informerObj := &kapi.Secret{}
	informerType := reflect.TypeOf(informerObj)
	informer, exists := f.informers[informerType]
	if exists {
		return informer
	}

	lw := f.customListerWatchers.GetListerWatcher(kapi.Resource("secrets"))
	if lw == nil {
		lw = &cache.ListWatch{
			ListFunc: func(options kapi.ListOptions) (runtime.Object, error) {
				return f.kubeClient.Core().Secrets(kapi.NamespaceAll).List(options)
			},
			WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
				return f.kubeClient.Core().Secrets(kapi.NamespaceAll).Watch(options)
			},
		}
	}

	informer = cache.NewSharedIndexInformer(
		lw,
		informerObj,
		f.defaultResync,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	f.informers[informerType] = informer

	return informer
}

func (f *secretInformer) Indexer() cache.Indexer {
	informer := f.Informer()
	return informer.GetIndexer()
}

// Lister is not implemented for Secret
func (f *secretInformer) Lister() interface{} {
	return nil
}

type ReplicationControllerInformer interface {
	Informer() cache.SharedIndexInformer
	Indexer() cache.Indexer
	Lister() *cache.StoreToReplicationControllerLister
}

type replicationControllerInformer struct {
	*sharedInformerFactory
}

func (f *replicationControllerInformer) Informer() cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	informerObj := &kapi.ReplicationController{}
	informerType := reflect.TypeOf(informerObj)
	informer, exists := f.informers[informerType]
	if exists {
		return informer
	}

	lw := f.customListerWatchers.GetListerWatcher(kapi.Resource("replicationcontrollers"))
	if lw == nil {
		lw = &cache.ListWatch{
			ListFunc: func(options kapi.ListOptions) (runtime.Object, error) {
				return f.kubeClient.Core().ReplicationControllers(kapi.NamespaceAll).List(options)
			},
			WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
				return f.kubeClient.Core().ReplicationControllers(kapi.NamespaceAll).Watch(options)
			},
		}
	}

	informer = cache.NewSharedIndexInformer(
		lw,
		informerObj,
		f.defaultResync,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	f.informers[informerType] = informer

	return informer
}

func (f *replicationControllerInformer) Indexer() cache.Indexer {
	informer := f.Informer()
	return informer.GetIndexer()
}

func (f *replicationControllerInformer) Lister() *cache.StoreToReplicationControllerLister {
	informer := f.Informer()
	return &cache.StoreToReplicationControllerLister{Indexer: informer.GetIndexer()}
}
