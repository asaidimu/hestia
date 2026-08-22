package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

type SettingDocumentView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Key                    string `anansi:"key"`
	Value                  any    `anansi:"value"`
}

type SettingOutput struct {
	Document SettingDocumentView `anansi:"document"`
}

type SettingsListOutput struct {
	Documents []SettingDocumentView `anansi:"documents"`
}

type MessageOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Message                string `anansi:"message"`
}