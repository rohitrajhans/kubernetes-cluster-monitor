package controller

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	log "github.com/sirupsen/logrus"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// controller object
type Controller struct {
	logger 		*log.Entry
	clientset 	kubernetes.Interface
	queue 		workqueue.RateLimitingInterface
	informer 	cache.SharedIndexInformer
	handler 	Handler
}

// function to start controller
func (c *Controller) Run(stopCh <- chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("Controller.Run: initiating")
	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
		return
	}
	c.logger.Info("Controller.Run: cache sync complete")

	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

func (c *Controller) runWorker() {
	log.Info("Controller.runWorker: starting")
	for c.processNextItem() {
		log.Info("Controller.runWorker: processing next item")
	}

	log.Info("Controller.runWorker: completed")
}

// function to process next item in the queue
func (c *Controller) processNextItem() bool {
	log.Info("Controller.processNextItem: start")
	// retrieve item from the queue
	key, quit := c.queue.Get()
	// if no item retreived, returns false
	if quit {
		return false
	}

	defer c.queue.Done(key)
	keyRaw := key.(string)
	// check if item exists
	// if it exists, resource has been added or updated
	// if it does not exist, the resource has been deleted
	// take appropriate action based on the `exists` bool
	item, exists, err := c.informer.GetIndexer().GetByKey(keyRaw)
	// c.logger.Println(item)

	// if there is an error in retrieving, try again for a certain number of times
	if err != nil {
		if c.queue.NumRequeues(key) < 5 {
			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", key, err)
			c.queue.AddRateLimited(key)
		} else {
			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", key, err)
			c.queue.Forget(key)
			utilruntime.HandleError(err)
		}
	}

	if !exists {
		// handler in case of object deletion was called before adding to queue
		c.logger.Infof("Controller.processNextItem: object deleted detected: %s", keyRaw)
		//c.handler.ObjectDeleted(item, c)
		c.queue.Forget(key)
	} else {
		// if resource has been added/ deleted, then call the appropriate handler
		c.logger.Infof("Controller.processNextItem: object created detected: %s", keyRaw)
		c.handler.ObjectCreated(item, c)
		c.queue.Forget(key)

        // c.logger.Infof("Check whether object has been updated")

		// oldKey, err := c.queue.Get()
		// if err {
		// 	c.logger.Infof("Controller.processNextItem: object just created")
		// 	return true
		// }

		// _, oldExists, error := c.informer.GetIndexer().GetByKey(keyRaw)

		// if error != nil {
		// 	if c.queue.NumRequeues(key) < 5 {
		// 		c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", oldKey, error)
		// 		c.queue.AddRateLimited(oldKey)
		// 	} else {
		// 		c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", oldKey, error)
		// 		c.queue.Forget(oldKey)
		// 		utilruntime.HandleError(error)
		// 	}
		// }

		// if oldExists {
		// 	c.logger.Infof("Controller.processNextItem: object not updated")
		// } else {
		// 	c.logger.Infof("Controller.processNextItem: object updated")
		// 	c.queue.Forget(oldKey)
		// }
	}

	return true

}
