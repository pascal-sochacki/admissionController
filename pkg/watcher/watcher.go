package watcher

import (
	"context"
	"log"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Watcher struct {
	Client kubernetes.Clientset
}

func (w Watcher) Start(ctx context.Context) error {
	watcher, err := w.Client.CoreV1().Namespaces().Watch(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	go func() {
		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}
				log.Printf(string(event.Type))
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}
