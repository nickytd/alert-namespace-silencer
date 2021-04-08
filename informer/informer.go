package informer

import (
	"fmt"
	"github.com/nickytd/alert-namespace-silencer/silencer"
	v1_ "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type NamespaceInformer struct {
	Cfg         *rest.Config
	AddQueue    *workqueue.Type
	DeleteQueue *workqueue.Type
	StopCh      <-chan struct{}
}

func (s *NamespaceInformer) RunNamespaceInformer(label string, silenceMatcherAttribute string) {
	client, err := kubernetes.NewForConfig(s.Cfg)
	if err != nil {
		klog.ErrorS(err, "error creating client")
	}

	informerFactory := informers.NewSharedInformerFactory(client, time.Minute*10)
	informer := informerFactory.Core().V1().Namespaces()
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(app interface{}) {
			namespace := app.(*v1_.Namespace).DeepCopyObject().(*v1_.Namespace)
			s.AddQueue.Add(namespace)
		},

		UpdateFunc: func(old, new interface{}) {
			namespaceOld := old.(*v1_.Namespace).DeepCopyObject().(*v1_.Namespace)
			namespaceNew := new.(*v1_.Namespace).DeepCopyObject().(*v1_.Namespace)
			if namespaceOld.ResourceVersion != namespaceNew.ResourceVersion {
				s.AddQueue.Add(namespaceNew)
			}
		},
		DeleteFunc: func(app interface{}) {
			namespace := app.(*v1_.Namespace).DeepCopyObject().(*v1_.Namespace)
			s.DeleteQueue.Add(namespace)
		},
	})

	go s.runAdd(label, silenceMatcherAttribute)
	go s.runDelete(silenceMatcherAttribute)

	informerFactory.Start(s.StopCh)
	informerFactory.WaitForCacheSync(s.StopCh)

	if ns, err := informer.Lister().List(labels.Everything()); err != nil {
		klog.ErrorS(err, "error listing namespaces")
	} else {
		for _, n := range ns {
			klog.V(4).InfoS("lister", "namespace", n.Name)
		}
	}

}

func (s *NamespaceInformer) runAdd(label string, silenceMatcherAttribute string) {
	for {
		item, _ := s.AddQueue.Get()
		n := item.(*v1_.Namespace)
		uid := fmt.Sprintf("%s", n.GetUID())
		klog.V(2).InfoS(
			"add queue",
			"name", n.Name,
			"id", uid,
			"silence matcher name", silenceMatcherAttribute,
		)

		//since the label is enable-alerts we remove silencer when present
		if flag, found := n.GetLabels()[label]; found && strings.ToLower(flag) == "true" {
			if silencer.RemoveSilencer(silenceMatcherAttribute, n.GetName()) {
				s.AddQueue.Done(item)
			}
		} else {
			if silencer.AddSilencer(silenceMatcherAttribute, n.GetName()) {
				s.AddQueue.Done(item)
			}
		}
	}
}

func (s *NamespaceInformer) runDelete(silenceMatcherAttribute string) {

	for {
		item, _ := s.DeleteQueue.Get()
		n := item.(*v1_.Namespace)
		uid := fmt.Sprintf("%s", n.GetUID())
		klog.V(2).InfoS(
			"delete queue",
			"name", n.Name,
			"id", uid,
			"silence matcher name", silenceMatcherAttribute,
		)

		if silencer.RemoveSilencer(silenceMatcherAttribute, n.GetName()) {
			s.DeleteQueue.Done(item)
		}
	}
}
