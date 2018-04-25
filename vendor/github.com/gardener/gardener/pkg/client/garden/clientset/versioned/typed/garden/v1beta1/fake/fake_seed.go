package fake

import (
	v1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSeeds implements SeedInterface
type FakeSeeds struct {
	Fake *FakeGardenV1beta1
}

var seedsResource = schema.GroupVersionResource{Group: "garden.sapcloud.io", Version: "v1beta1", Resource: "seeds"}

var seedsKind = schema.GroupVersionKind{Group: "garden.sapcloud.io", Version: "v1beta1", Kind: "Seed"}

// Get takes name of the seed, and returns the corresponding seed object, and an error if there is any.
func (c *FakeSeeds) Get(name string, options v1.GetOptions) (result *v1beta1.Seed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(seedsResource, name), &v1beta1.Seed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Seed), err
}

// List takes label and field selectors, and returns the list of Seeds that match those selectors.
func (c *FakeSeeds) List(opts v1.ListOptions) (result *v1beta1.SeedList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(seedsResource, seedsKind, opts), &v1beta1.SeedList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.SeedList{}
	for _, item := range obj.(*v1beta1.SeedList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested seeds.
func (c *FakeSeeds) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(seedsResource, opts))
}

// Create takes the representation of a seed and creates it.  Returns the server's representation of the seed, and an error, if there is any.
func (c *FakeSeeds) Create(seed *v1beta1.Seed) (result *v1beta1.Seed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(seedsResource, seed), &v1beta1.Seed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Seed), err
}

// Update takes the representation of a seed and updates it. Returns the server's representation of the seed, and an error, if there is any.
func (c *FakeSeeds) Update(seed *v1beta1.Seed) (result *v1beta1.Seed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(seedsResource, seed), &v1beta1.Seed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Seed), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSeeds) UpdateStatus(seed *v1beta1.Seed) (*v1beta1.Seed, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(seedsResource, "status", seed), &v1beta1.Seed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Seed), err
}

// Delete takes name of the seed and deletes it. Returns an error if one occurs.
func (c *FakeSeeds) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(seedsResource, name), &v1beta1.Seed{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSeeds) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(seedsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.SeedList{})
	return err
}

// Patch applies the patch and returns the patched seed.
func (c *FakeSeeds) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Seed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(seedsResource, name, data, subresources...), &v1beta1.Seed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Seed), err
}
