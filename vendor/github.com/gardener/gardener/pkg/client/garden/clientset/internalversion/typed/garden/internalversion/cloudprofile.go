package internalversion

import (
	garden "github.com/gardener/gardener/pkg/apis/garden"
	scheme "github.com/gardener/gardener/pkg/client/garden/clientset/internalversion/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CloudProfilesGetter has a method to return a CloudProfileInterface.
// A group's client should implement this interface.
type CloudProfilesGetter interface {
	CloudProfiles() CloudProfileInterface
}

// CloudProfileInterface has methods to work with CloudProfile resources.
type CloudProfileInterface interface {
	Create(*garden.CloudProfile) (*garden.CloudProfile, error)
	Update(*garden.CloudProfile) (*garden.CloudProfile, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*garden.CloudProfile, error)
	List(opts v1.ListOptions) (*garden.CloudProfileList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *garden.CloudProfile, err error)
	CloudProfileExpansion
}

// cloudProfiles implements CloudProfileInterface
type cloudProfiles struct {
	client rest.Interface
}

// newCloudProfiles returns a CloudProfiles
func newCloudProfiles(c *GardenClient) *cloudProfiles {
	return &cloudProfiles{
		client: c.RESTClient(),
	}
}

// Get takes name of the cloudProfile, and returns the corresponding cloudProfile object, and an error if there is any.
func (c *cloudProfiles) Get(name string, options v1.GetOptions) (result *garden.CloudProfile, err error) {
	result = &garden.CloudProfile{}
	err = c.client.Get().
		Resource("cloudprofiles").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CloudProfiles that match those selectors.
func (c *cloudProfiles) List(opts v1.ListOptions) (result *garden.CloudProfileList, err error) {
	result = &garden.CloudProfileList{}
	err = c.client.Get().
		Resource("cloudprofiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cloudProfiles.
func (c *cloudProfiles) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("cloudprofiles").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a cloudProfile and creates it.  Returns the server's representation of the cloudProfile, and an error, if there is any.
func (c *cloudProfiles) Create(cloudProfile *garden.CloudProfile) (result *garden.CloudProfile, err error) {
	result = &garden.CloudProfile{}
	err = c.client.Post().
		Resource("cloudprofiles").
		Body(cloudProfile).
		Do().
		Into(result)
	return
}

// Update takes the representation of a cloudProfile and updates it. Returns the server's representation of the cloudProfile, and an error, if there is any.
func (c *cloudProfiles) Update(cloudProfile *garden.CloudProfile) (result *garden.CloudProfile, err error) {
	result = &garden.CloudProfile{}
	err = c.client.Put().
		Resource("cloudprofiles").
		Name(cloudProfile.Name).
		Body(cloudProfile).
		Do().
		Into(result)
	return
}

// Delete takes name of the cloudProfile and deletes it. Returns an error if one occurs.
func (c *cloudProfiles) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("cloudprofiles").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *cloudProfiles) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("cloudprofiles").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched cloudProfile.
func (c *cloudProfiles) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *garden.CloudProfile, err error) {
	result = &garden.CloudProfile{}
	err = c.client.Patch(pt).
		Resource("cloudprofiles").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
