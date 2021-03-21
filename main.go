package main

import (
	"flag"
	"github.com/nickytd/alert-namespace-silencer/informer"
	"github.com/nickytd/alert-namespace-silencer/silencer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"

	//need to run when oath token kubeconfig is supplied
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	kubeconfig              string
	alertmanagerUrl         string
	label                   string
	silenceMatcherAttribute string
)

func main() {

	klog.InitFlags(nil)
	klog.InfoS("starting alert-namespace-silencer")

	flag.StringVar(
		&kubeconfig,
		"kubeconfig",
		defaultKubeconfig(),
		"path to kubeconfig")

	flag.StringVar(
		&label,
		"enable-label",
		"enable-alerts=true",
		"namespace label to creating alert silences")

	flag.StringVar(
		&alertmanagerUrl,
		"alertmanager-url",
		"http://alertmanager-operated:9093",
		"Url of the alertmanager")

	flag.StringVar(
		&silenceMatcherAttribute,
		"silence-matcher-attribute",
		"namespace",
		"silence matcher attribute")

	flag.Parse()

	stopCh := initStopCh()

	namespaceInformer := informer.NamespaceInformer{
		StopCh:      stopCh,
		Cfg:         initClientSet(kubeconfig),
		AddQueue:    workqueue.NewNamed("addQueue"),
		DeleteQueue: workqueue.NewNamed("deleteQueue"),
	}

	if alertmanagerURL, err := url.Parse(alertmanagerUrl); err != nil {
		klog.ErrorS(err, "invalid alertmanager url")
	} else {
		silencer.InitAlertManager(*alertmanagerURL)
		namespaceInformer.RunNamespaceInformer(label, silenceMatcherAttribute)
	}

	klog.Info("running ...")
	<-stopCh
	klog.Info("exiting ...")
}

func defaultKubeconfig() string {
	fileName := os.Getenv("KUBECONFIG")
	if fileName != "" {
		return fileName
	}
	home, err := os.UserHomeDir()
	if err != nil {
		klog.Warningf("failed to get home directory: %s", err.Error())
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

func initClientSet(kubeconfig string) *rest.Config {
	var config *rest.Config
	var err error

	klog.V(4).Infof("kubeconfig %s", kubeconfig)

	if config, err = rest.InClusterConfig(); err != nil && config == nil {
		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			klog.Fatalf("error creating config %s", err.Error())
		}
	}
	return config
}

func initStopCh() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		signal := <-c
		klog.InfoS("signal", "received", signal.String())
		close(stop)
		<-c
		os.Exit(1)
	}()
	return stop
}
