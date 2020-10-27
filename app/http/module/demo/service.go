package demo

type Service struct {
	repository *Repository
}

func NewService() *Service {
	repository := NewRepository()
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetUsers() []UserModel {
	ids := s.repository.GetUserIds()
	return s.repository.GetUserByIds(ids)
}
