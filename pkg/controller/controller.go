package controller

import (
    "fmt"
    "kube-controlller-the-hard-way/pkg/utils"
    "time"

    "github.com/sirupsen/logrus"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/apimachinery/pkg/watch"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/util/workqueue"

    apiv1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const maxRetries = 5

// Controller object
type Controller struct {
    logger    *logrus.Entry
    clientset kubernetes.Interface
    queue     workqueue.RateLimitingInterface
    informer  cache.SharedIndexInformer
    // eventHandler handlers.Handler
}

func NewPodsController(client kubernetes.Interface) *Controller {
    queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
    informer := cache.NewSharedIndexInformer(
        &cache.ListWatch{
            ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
                return client.CoreV1().Pods(metav1.NamespaceAll).List(options)
            },
            WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
                return client.CoreV1().Pods(metav1.NamespaceAll).Watch(options)
            },
        },
        &apiv1.Pod{},
        0, //Skip resync
        cache.Indexers{},
    )

    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            key, err := cache.MetaNamespaceKeyFunc(obj)
            if err == nil {
                queue.Add(key)
            }
        },
        DeleteFunc: func(obj interface{}) {
            key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
            if err == nil {
                queue.Add(key)
            }
        },
    })
    return &Controller{
        logger:    logrus.WithField("pkg", "pods"),
        clientset: client,
        informer:  informer,
        queue:     queue,
        // eventHandler: eventHandler,
    }
}

// Run will start the controller.
// StopCh channel is used to send interrupt signal to stop it.
func (c *Controller) Run(stopCh <-chan struct{}) {
    // don't let panics crash the process
    defer utilruntime.HandleCrash()
    // make sure the work queue is shutdown which will trigger workers to end
    defer c.queue.ShutDown()

    c.logger.Info("Starting the dummy controller...")
    go c.informer.Run(stopCh)

    // wait for the caches to synchronize before starting the worker
    if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
        utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync "))
    }
    c.logger.Info("dummy controller synced and ready")

    // runWorker will loop until "something bad" happens.  The .Until will
    // then rekick the worker after one second
    wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *Controller) runWorker() {
    // processNextWorkItem will automatically wait until there's work available
    for c.processNextItem() {
        // continue looping
    }
}

func (c *Controller) processNextItem() bool {
    // pull the next work item from queue.  It should be a key we use to lookup
    // something in a cache
    key, quit := c.queue.Get()
    if quit {
        return false
    }
    // you always have to indicate to the queue that you've completed a piece of
    // work
    defer c.queue.Done(key)

    // do your work on the key.
    err := c.processItem(key.(string))

    if err == nil {
        // No error, tell the queue to stop tracking history
        c.queue.Forget(key)
    } else if c.queue.NumRequeues(key) < maxRetries {
        c.logger.Errorf("Error processing %s (will retry): %v", key, err)
        // requeue the item to work on later
        c.queue.AddRateLimited(key)
    } else {
        // err != nil and too many retries
        c.logger.Errorf("Error processing %s (giving up): %v", key, err)
        c.queue.Forget(key)
        utilruntime.HandleError(err)
    }

    return true
}

func (c *Controller) processItem(key string) error {
    c.logger.Infof("Processing change on Pod %s", key)

    obj, exists, err := c.informer.GetIndexer().GetByKey(key)

    if err != nil {
        return fmt.Errorf("Error fetching object with key %s from store: %v ", key, err)
    }

    if !exists {
        // c.eventHandler.ObjectDeleted(obj)
        c.logger.Infof("Pod %s have been deleted ", key)

        return nil
    }
    objectMeta := utils.GetObjectMetaData(obj)
    // c.eventHandler.ObjectCreated(obj)
    c.logger.Infof("Pod change have been processed: %s: %#v ", key, objectMeta.Name)
    return nil
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
    return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
    return c.informer.LastSyncResourceVersion()
}
