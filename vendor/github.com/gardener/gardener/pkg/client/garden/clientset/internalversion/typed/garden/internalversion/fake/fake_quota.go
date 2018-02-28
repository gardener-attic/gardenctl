package fake

import (
	garden "github.com/gardener/gardener/pkg/apis/garden"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeQuotas implements QuotaInterface
type FakeQuotas struct {
	Fake *FakeGarden
	ns   string
}

var quotasResource = schema.GroupVersionResource{Group: "garden.sapcloud.io", Version: "", Resource: "quotas"}

var quotasKind = schema.GroupVersionKind{Group: "garden.sapcloud.io", Version: "", Kind: "Quota"}

// Get takes name of the quota, and returns the corresponding quota object, and an error if there is any.
func (c *FakeQuotas) Get(name string, options v1.GetOptions) (result *garden.Quota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(quotasResource, c.ns, name), &garden.Quota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*garden.Quota), err
}

// List takes label and field selectors, and returns the list of Quotas that match those selectors.
func (c *FakeQuotas) List(opts v1.ListOptions) (result *garden.QuotaList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(quotasResource, quotasKind, c.ns, opts), &garden.QuotaList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &garden.QuotaList{}
	for _, item := range obj.(*garden.QuotaList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested quotas.
func (c *FakeQuotas) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(quotasResource, c.ns, opts))

}

// Create takes the representation of a quota and creates it.  Returns the server's representation of the quota, and an error, if there is any.
func (c *FakeQuotas) Create(quota *garden.Quota) (result *garden.Quota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(quotasResource, c.ns, quota), &garden.Quota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*garden.Quota), err
}

// Update takes the representation of a quota and updates it. Returns the server's representation of the quota, and an error, if there is any.
func (c *FakeQuotas) Update(quota *garden.Quota) (result *garden.Quota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(quotasResource, c.ns, quota), &garden.Quota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*garden.Quota), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeQuotas) UpdateStatus(quota *garden.Quota) (*garden.Quota, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(quotasResource, "status", c.ns, quota), &garden.Quota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*garden.Quota), err
}

// Delete takes name of the quota and deletes it. Returns an error if one occurs.
func (c *FakeQuotas) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(quotasResource, c.ns, name), &garden.Quota{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeQuotas) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(quotasResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &garden.QuotaList{})
	return err
}

// Patch applies the patch and returns the patched quota.
func (c *FakeQuotas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *garden.Quota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(quotasResource, c.ns, name, data, subresources...), &garden.Quota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*garden.Quota), err
}
