package main

import "k8s.io/klog/v2"

func main() {

	klog.InitFlags(nil)
	klog.InfoS("starting alert-namespace-silencer")

}
