package models

type Word struct {
	ID            uint    `gorm:"primaryKey"`
	MainWord      *string `gorm:"null"`
	TranslateWord *string `gorm:"null"`
}
