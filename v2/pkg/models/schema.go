package models

// SchemaVersion migration
type SchemaVersion struct {
	Base
	Service string `gorm:"unique_index:service_version"`
	Version int    `gorm:"unique_index:service_version"`
}
