package framework

// NewInstance define a instantiate function for making a new service
type NewInstance func(...interface{}) (interface{}, error)

// ServiceProvider define a service provider should implement
type ServiceProvider interface {
	// Register registe a new function for make a service instance
	Register(Container) NewInstance
	// Boot will called when the service instantiate
	Boot(Container) error
	// IsDefer define whether the service instantiate when first make or register
	IsDefer() bool
	// Params define the necessary params for NewInstance
	Params() []interface{}
	/// Name define the name for this service
	Name() string
}
