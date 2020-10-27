package contract

const IDKey = "hade:id"

type IDService interface {
	NewID() string
}
