package authorization

type ServiceOption func(*Service)

func WithMFAGatesDisabled() ServiceOption {
	return func(service *Service) {
		service.disableMFAGates = true
	}
}
