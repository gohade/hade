package demo

type Repository struct {
}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) GetUserIds() []int {
	return []int{1, 2}
}

func (r *Repository) GetUserByIds([]int) []UserModel {
	return []UserModel{
		{
			UserId: 1,
			Name:   "foo",
			Age:    11,
		},
		{
			UserId: 2,
			Name:   "bar",
			Age:    12,
		},
	}
}
