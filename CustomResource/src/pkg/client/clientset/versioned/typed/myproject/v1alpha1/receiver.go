/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/cisco/CustomResource/src/pkg/apis/myproject/v1alpha1"
	scheme "github.com/cisco/CustomResource/src/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ReceiversGetter has a method to return a ReceiverInterface.
// A group's client should implement this interface.
type ReceiversGetter interface {
	Receivers(namespace string) ReceiverInterface
}

// ReceiverInterface has methods to work with Receiver resources.
type ReceiverInterface interface {
	Create(ctx context.Context, receiver *v1alpha1.Receiver, opts v1.CreateOptions) (*v1alpha1.Receiver, error)
	Update(ctx context.Context, receiver *v1alpha1.Receiver, opts v1.UpdateOptions) (*v1alpha1.Receiver, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Receiver, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ReceiverList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Receiver, err error)
	ReceiverExpansion
}

// receivers implements ReceiverInterface
type receivers struct {
	client rest.Interface
	ns     string
}

// newReceivers returns a Receivers
func newReceivers(c *SampleprojectV1alpha1Client, namespace string) *receivers {
	return &receivers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the receiver, and returns the corresponding receiver object, and an error if there is any.
func (c *receivers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Receiver, err error) {
	result = &v1alpha1.Receiver{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("receivers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Receivers that match those selectors.
func (c *receivers) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ReceiverList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ReceiverList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("receivers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested receivers.
func (c *receivers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("receivers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a receiver and creates it.  Returns the server's representation of the receiver, and an error, if there is any.
func (c *receivers) Create(ctx context.Context, receiver *v1alpha1.Receiver, opts v1.CreateOptions) (result *v1alpha1.Receiver, err error) {
	result = &v1alpha1.Receiver{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("receivers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(receiver).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a receiver and updates it. Returns the server's representation of the receiver, and an error, if there is any.
func (c *receivers) Update(ctx context.Context, receiver *v1alpha1.Receiver, opts v1.UpdateOptions) (result *v1alpha1.Receiver, err error) {
	result = &v1alpha1.Receiver{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("receivers").
		Name(receiver.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(receiver).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the receiver and deletes it. Returns an error if one occurs.
func (c *receivers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("receivers").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *receivers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("receivers").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched receiver.
func (c *receivers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Receiver, err error) {
	result = &v1alpha1.Receiver{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("receivers").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
