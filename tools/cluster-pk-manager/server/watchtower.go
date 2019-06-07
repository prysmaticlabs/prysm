package main

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var queryInterval = 3 * time.Second
var namespace = "beacon-chain"
var podSelector = "component=validator"

type watchtower struct {
	db     *db
	client *kubernetes.Clientset
}

func newWatchtower(db *db) *watchtower {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	client := kubernetes.NewForConfigOrDie(config)
	return &watchtower{db, client}
}

// WatchPods for termination, update allocations
func (wt *watchtower) WatchPods() {
	for {
		time.Sleep(queryInterval)
		if err := wt.queryPodsAndUpdateDB(); err != nil {
			log.WithField("error", err).Error("Failed to update pods")
		}
	}
}

// Query k8s pods for existence.
func (wt *watchtower) queryPodsAndUpdateDB() error {
	ctx := context.Background()
	podNames, err := wt.db.AllocatedPodNames(ctx)
	if err != nil {
		return err
	}
	pList, err := wt.client.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: podSelector})
	if err != nil {
		return err
	}
	podExist := make(map[string]bool)
	for _, p := range pList.Items {
		if p.Status.Phase == v1.PodRunning || p.Status.Phase == v1.PodPending {
			podExist[p.Name] = true
		} else {
			log.Debugf("ignoring pod with phase %s", p.Status.Phase)
		}
	}

	for _, p := range podNames {
		if !podExist[p] {
			log.WithField("pod", p).Debug("Removing assignment from dead pod")
			if err := wt.db.RemovePKAssignment(ctx, p); err != nil {
				return err
			}
		}
	}

	return nil
}
