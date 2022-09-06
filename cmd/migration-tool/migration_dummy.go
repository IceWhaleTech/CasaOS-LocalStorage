package main

import interfaces "github.com/IceWhaleTech/CasaOS-Common"

type migrationTool1 struct{}

func (u *migrationTool1) IsMigrationNeeded() (bool, error) {
	return false, nil
}

func (u *migrationTool1) PreMigrate() error {
	return nil
}

func (u *migrationTool1) Migrate() error {
	return nil
}

func (u *migrationTool1) PostMigrate() error {
	return nil
}

func NewMigrationToolDummy() interfaces.MigrationTool {
	return &migrationTool1{}
}
