// Code generated by solo-kit. DO NOT EDIT.

package v1

import (
	"sync"
	"time"

	istio_networking_v1alpha3 "github.com/solo-io/sg/pkg/api/external/istio/networking/v1alpha3"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/errors"
	"github.com/solo-io/solo-kit/pkg/utils/errutils"
)

var (
	mRoutingSnapshotIn  = stats.Int64("routing.sg.solo.io/snap_emitter/snap_in", "The number of snapshots in", "1")
	mRoutingSnapshotOut = stats.Int64("routing.sg.solo.io/snap_emitter/snap_out", "The number of snapshots out", "1")

	routingsnapshotInView = &view.View{
		Name:        "routing.sg.solo.io_snap_emitter/snap_in",
		Measure:     mRoutingSnapshotIn,
		Description: "The number of snapshots updates coming in",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{},
	}
	routingsnapshotOutView = &view.View{
		Name:        "routing.sg.solo.io/snap_emitter/snap_out",
		Measure:     mRoutingSnapshotOut,
		Description: "The number of snapshots updates going out",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{},
	}
)

func init() {
	view.Register(routingsnapshotInView, routingsnapshotOutView)
}

type RoutingEmitter interface {
	Register() error
	RoutingRule() RoutingRuleClient
	DestinationRule() istio_networking_v1alpha3.DestinationRuleClient
	VirtualService() istio_networking_v1alpha3.VirtualServiceClient
	Snapshots(watchNamespaces []string, opts clients.WatchOpts) (<-chan *RoutingSnapshot, <-chan error, error)
}

func NewRoutingEmitter(routingRuleClient RoutingRuleClient, destinationRuleClient istio_networking_v1alpha3.DestinationRuleClient, virtualServiceClient istio_networking_v1alpha3.VirtualServiceClient) RoutingEmitter {
	return NewRoutingEmitterWithEmit(routingRuleClient, destinationRuleClient, virtualServiceClient, make(chan struct{}))
}

func NewRoutingEmitterWithEmit(routingRuleClient RoutingRuleClient, destinationRuleClient istio_networking_v1alpha3.DestinationRuleClient, virtualServiceClient istio_networking_v1alpha3.VirtualServiceClient, emit <-chan struct{}) RoutingEmitter {
	return &routingEmitter{
		routingRule:     routingRuleClient,
		destinationRule: destinationRuleClient,
		virtualService:  virtualServiceClient,
		forceEmit:       emit,
	}
}

type routingEmitter struct {
	forceEmit       <-chan struct{}
	routingRule     RoutingRuleClient
	destinationRule istio_networking_v1alpha3.DestinationRuleClient
	virtualService  istio_networking_v1alpha3.VirtualServiceClient
}

func (c *routingEmitter) Register() error {
	if err := c.routingRule.Register(); err != nil {
		return err
	}
	if err := c.destinationRule.Register(); err != nil {
		return err
	}
	if err := c.virtualService.Register(); err != nil {
		return err
	}
	return nil
}

func (c *routingEmitter) RoutingRule() RoutingRuleClient {
	return c.routingRule
}

func (c *routingEmitter) DestinationRule() istio_networking_v1alpha3.DestinationRuleClient {
	return c.destinationRule
}

func (c *routingEmitter) VirtualService() istio_networking_v1alpha3.VirtualServiceClient {
	return c.virtualService
}

func (c *routingEmitter) Snapshots(watchNamespaces []string, opts clients.WatchOpts) (<-chan *RoutingSnapshot, <-chan error, error) {
	errs := make(chan error)
	var done sync.WaitGroup
	ctx := opts.Ctx
	/* Create channel for RoutingRule */
	type routingRuleListWithNamespace struct {
		list      RoutingRuleList
		namespace string
	}
	routingRuleChan := make(chan routingRuleListWithNamespace)
	/* Create channel for DestinationRule */
	type destinationRuleListWithNamespace struct {
		list      istio_networking_v1alpha3.DestinationRuleList
		namespace string
	}
	destinationRuleChan := make(chan destinationRuleListWithNamespace)
	/* Create channel for VirtualService */
	type virtualServiceListWithNamespace struct {
		list      istio_networking_v1alpha3.VirtualServiceList
		namespace string
	}
	virtualServiceChan := make(chan virtualServiceListWithNamespace)

	for _, namespace := range watchNamespaces {
		/* Setup namespaced watch for RoutingRule */
		routingRuleNamespacesChan, routingRuleErrs, err := c.routingRule.Watch(namespace, opts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "starting RoutingRule watch")
		}

		done.Add(1)
		go func(namespace string) {
			defer done.Done()
			errutils.AggregateErrs(ctx, errs, routingRuleErrs, namespace+"-destinationrules")
		}(namespace)
		/* Setup namespaced watch for DestinationRule */
		destinationRuleNamespacesChan, destinationRuleErrs, err := c.destinationRule.Watch(namespace, opts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "starting DestinationRule watch")
		}

		done.Add(1)
		go func(namespace string) {
			defer done.Done()
			errutils.AggregateErrs(ctx, errs, destinationRuleErrs, namespace+"-destinationrules")
		}(namespace)
		/* Setup namespaced watch for VirtualService */
		virtualServiceNamespacesChan, virtualServiceErrs, err := c.virtualService.Watch(namespace, opts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "starting VirtualService watch")
		}

		done.Add(1)
		go func(namespace string) {
			defer done.Done()
			errutils.AggregateErrs(ctx, errs, virtualServiceErrs, namespace+"-virtualservices")
		}(namespace)

		/* Watch for changes and update snapshot */
		go func(namespace string) {
			for {
				select {
				case <-ctx.Done():
					return
				case routingRuleList := <-routingRuleNamespacesChan:
					select {
					case <-ctx.Done():
						return
					case routingRuleChan <- routingRuleListWithNamespace{list: routingRuleList, namespace: namespace}:
					}
				case destinationRuleList := <-destinationRuleNamespacesChan:
					select {
					case <-ctx.Done():
						return
					case destinationRuleChan <- destinationRuleListWithNamespace{list: destinationRuleList, namespace: namespace}:
					}
				case virtualServiceList := <-virtualServiceNamespacesChan:
					select {
					case <-ctx.Done():
						return
					case virtualServiceChan <- virtualServiceListWithNamespace{list: virtualServiceList, namespace: namespace}:
					}
				}
			}
		}(namespace)
	}

	snapshots := make(chan *RoutingSnapshot)
	go func() {
		originalSnapshot := RoutingSnapshot{}
		currentSnapshot := originalSnapshot.Clone()
		timer := time.NewTicker(time.Second * 1)
		sync := func() {
			if originalSnapshot.Hash() == currentSnapshot.Hash() {
				return
			}

			stats.Record(ctx, mRoutingSnapshotOut.M(1))
			originalSnapshot = currentSnapshot.Clone()
			sentSnapshot := currentSnapshot.Clone()
			snapshots <- &sentSnapshot
		}

		/* TODO (yuval-k): figure out how to make this work to avoid a stale snapshot.
		   		// construct the first snapshot from all the configs that are currently there
		   		// that guarantees that the first snapshot contains all the data.
		   		for range watchNamespaces {
		      routingRuleNamespacedList := <- routingRuleChan
		      currentSnapshot.Destinationrules.Clear(routingRuleNamespacedList.namespace)
		      routingRuleList := routingRuleNamespacedList.list
		   	currentSnapshot.Destinationrules.Add(routingRuleList...)
		      destinationRuleNamespacedList := <- destinationRuleChan
		      currentSnapshot.Destinationrules.Clear(destinationRuleNamespacedList.namespace)
		      destinationRuleList := destinationRuleNamespacedList.list
		   	currentSnapshot.Destinationrules.Add(destinationRuleList...)
		      virtualServiceNamespacedList := <- virtualServiceChan
		      currentSnapshot.Virtualservices.Clear(virtualServiceNamespacedList.namespace)
		      virtualServiceList := virtualServiceNamespacedList.list
		   	currentSnapshot.Virtualservices.Add(virtualServiceList...)
		   		}
		*/

		for {
			record := func() { stats.Record(ctx, mRoutingSnapshotIn.M(1)) }

			select {
			case <-timer.C:
				sync()
			case <-ctx.Done():
				close(snapshots)
				done.Wait()
				close(errs)
				return
			case <-c.forceEmit:
				sentSnapshot := currentSnapshot.Clone()
				snapshots <- &sentSnapshot
			case routingRuleNamespacedList := <-routingRuleChan:
				record()

				namespace := routingRuleNamespacedList.namespace
				routingRuleList := routingRuleNamespacedList.list

				currentSnapshot.Destinationrules.Clear(namespace)
				currentSnapshot.Destinationrules.Add(routingRuleList...)
			case destinationRuleNamespacedList := <-destinationRuleChan:
				record()

				namespace := destinationRuleNamespacedList.namespace
				destinationRuleList := destinationRuleNamespacedList.list

				currentSnapshot.Destinationrules.Clear(namespace)
				currentSnapshot.Destinationrules.Add(destinationRuleList...)
			case virtualServiceNamespacedList := <-virtualServiceChan:
				record()

				namespace := virtualServiceNamespacedList.namespace
				virtualServiceList := virtualServiceNamespacedList.list

				currentSnapshot.Virtualservices.Clear(namespace)
				currentSnapshot.Virtualservices.Add(virtualServiceList...)
			}
		}
	}()
	return snapshots, errs, nil
}
