package contactfw

import (
	"context"
	"errors"
)

// CoreDelegatedClient adapts the generated or otherwise typed Core internal
// Contact binding to the Framework Store. Its dependency is the six-operation
// typed interface; arbitrary HTTP methods, endpoints, headers, and bodies
// cannot enter this boundary.
type CoreDelegatedClient struct{ binding DelegatedContactClient }

func NewCoreDelegatedClient(binding DelegatedContactClient) (*CoreDelegatedClient, error) {
	if binding == nil {
		return nil, errors.New("contact delegated binding is required")
	}
	return &CoreDelegatedClient{binding: binding}, nil
}

func (c *CoreDelegatedClient) Create(ctx context.Context, input CreateContactInput) (*Contact, error) {
	binding, err := c.requireBinding()
	if err != nil {
		return nil, err
	}
	item, err := binding.Create(ctx, input)
	return item, normalizeDelegatedError(err)
}
func (c *CoreDelegatedClient) Get(ctx context.Context, input GetContactInput) (*Contact, error) {
	binding, err := c.requireBinding()
	if err != nil {
		return nil, err
	}
	item, err := binding.Get(ctx, input)
	return item, normalizeDelegatedError(err)
}
func (c *CoreDelegatedClient) Update(ctx context.Context, input UpdateContactInput) (*Contact, error) {
	binding, err := c.requireBinding()
	if err != nil {
		return nil, err
	}
	item, err := binding.Update(ctx, input)
	return item, normalizeDelegatedError(err)
}
func (c *CoreDelegatedClient) ListByCustomer(ctx context.Context, input ListByCustomerInput) (ContactPage, error) {
	binding, err := c.requireBinding()
	if err != nil {
		return ContactPage{}, err
	}
	page, err := binding.ListByCustomer(ctx, input)
	return page, normalizeDelegatedError(err)
}
func (c *CoreDelegatedClient) ResolveIdentity(ctx context.Context, input ResolveContactIdentityInput) (*ContactIdentityResolution, error) {
	binding, err := c.requireBinding()
	if err != nil {
		return nil, err
	}
	item, err := binding.ResolveIdentity(ctx, input)
	return item, normalizeDelegatedError(err)
}
func (c *CoreDelegatedClient) BindIdentity(ctx context.Context, input BindContactIdentityInput) (*ContactIdentity, error) {
	binding, err := c.requireBinding()
	if err != nil {
		return nil, err
	}
	item, err := binding.BindIdentity(ctx, input)
	return item, normalizeDelegatedError(err)
}

func (c *CoreDelegatedClient) requireBinding() (DelegatedContactClient, error) {
	if c == nil || c.binding == nil {
		return nil, NewError(CodeDelegateUnavailable, errors.New("contact delegated binding is unavailable"))
	}
	return c.binding, nil
}

func normalizeDelegatedError(err error) error {
	if err == nil || CodeOf(err) != "" {
		return err
	}
	return NewError(CodeDelegateUnavailable, err)
}

var _ DelegatedContactClient = (*CoreDelegatedClient)(nil)
