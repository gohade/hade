package gen

import "gorm.io/gorm"

type StudentModel struct {
	gorm.Model
	Name    string `gorm:"name,not null"`
	Age     uint   `gorm:"age,not null"`
	ClassID uint   `gorm:"class_id,not null"`
}
