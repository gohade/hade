package gen

type StudentModel struct {
	ID      int64  `gorm:"id,primary_key"  json:"id,omitempty"`
	Name    string `gorm:"name,not null" json:"name,omitempty"`
	Age     uint   `gorm:"age,not null" json:"age,omitempty"`
	ClassID uint   `gorm:"class_id,not null" json:"class_id,omitempty"`
}

func (StudentModel) TableName() string {
	return "student"
}
